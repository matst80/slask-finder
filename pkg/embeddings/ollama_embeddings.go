package embeddings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/matst80/slask-finder/pkg/types"
)

const (
	// Ollama API endpoint
	ollamaEmbeddingEndpoint = "http://10.10.10.100:11434/api/embeddings"
	// Model to use for embeddings
	defaultEmbeddingModel = "mxbai-embed-large"
)

// OllamaEmbeddingRequest represents the request body for Ollama embeddings API
type OllamaEmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// OllamaEmbeddingResponse represents the response from Ollama embeddings API
type OllamaEmbeddingResponse struct {
	Embedding []float64 `json:"embedding"`
}

// OllamaEmbeddingsEngine implements the types.EmbeddingsEngine interface
// using Ollama's HTTP API for generating embeddings
type OllamaEmbeddingsEngine struct {
	Model        string
	ApiEndpoints []string
	HttpClient   *http.Client
	counter      uint32 // Used for round-robin endpoint selection

	// For backward compatibility
	ApiEndpoint string
}

// NewOllamaEmbeddingsEngine creates a new instance of OllamaEmbeddingsEngine
// with default configuration
func NewOllamaEmbeddingsEngine() *OllamaEmbeddingsEngine {
	return &OllamaEmbeddingsEngine{
		Model:        defaultEmbeddingModel,
		ApiEndpoints: []string{ollamaEmbeddingEndpoint},
		ApiEndpoint:  ollamaEmbeddingEndpoint, // For backward compatibility
		HttpClient:   &http.Client{},
		counter:      0,
	}
}

// NewOllamaEmbeddingsEngineWithConfig creates a new instance of OllamaEmbeddingsEngine
// with custom configuration
func NewOllamaEmbeddingsEngineWithConfig(model, endpoint string) *OllamaEmbeddingsEngine {
	if model == "" {
		model = defaultEmbeddingModel
	}
	if endpoint == "" {
		endpoint = ollamaEmbeddingEndpoint
	}

	return &OllamaEmbeddingsEngine{
		Model:        model,
		ApiEndpoints: []string{endpoint},
		ApiEndpoint:  endpoint, // For backward compatibility
		HttpClient:   &http.Client{},
		counter:      0,
	}
}

// NewOllamaEmbeddingsEngineWithMultipleEndpoints creates a new instance of OllamaEmbeddingsEngine
// with multiple API endpoints for round-robin load balancing
func NewOllamaEmbeddingsEngineWithMultipleEndpoints(model string, endpoints ...string) *OllamaEmbeddingsEngine {
	if model == "" {
		model = defaultEmbeddingModel
	}

	// If no endpoints provided, use the default
	if len(endpoints) == 0 {
		endpoints = []string{ollamaEmbeddingEndpoint}
	}

	return &OllamaEmbeddingsEngine{
		Model:        model,
		ApiEndpoints: endpoints,
		ApiEndpoint:  endpoints[0], // For backward compatibility
		HttpClient:   &http.Client{},
		counter:      0,
	}
}

// GenerateEmbeddings implements EmbeddingsEngine.GenerateEmbeddings
// It generates embeddings for the given text using Ollama API
func (o *OllamaEmbeddingsEngine) GenerateEmbeddings(text string) (types.Embeddings, error) {
	// Prepare the request body
	reqBody := OllamaEmbeddingRequest{
		Model:  o.Model,
		Prompt: text,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	// Select an endpoint using round-robin
	var endpoint string

	// Use ApiEndpoints if available (for round-robin), otherwise fall back to single ApiEndpoint
	if len(o.ApiEndpoints) > 0 {
		// Get the next index using atomic counter for thread safety
		idx := atomic.AddUint32(&o.counter, 1) % uint32(len(o.ApiEndpoints))
		endpoint = o.ApiEndpoints[idx]
	} else {
		endpoint = o.ApiEndpoint
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send the request
	resp, err := o.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request to Ollama API: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-OK response from Ollama API: %d", resp.StatusCode)
	}

	// Parse the response
	var ollamaResp OllamaEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("error decoding response from Ollama API: %w", err)
	}

	// Convert float64 embeddings to float32 for the types.Embeddings interface
	float32Embeddings := make(types.Embeddings, len(ollamaResp.Embedding))
	for i, val := range ollamaResp.Embedding {
		float32Embeddings[i] = float32(val)
	}

	// Return the embeddings
	return float32Embeddings, nil
}

// GenerateEmbeddingsFromItem implements EmbeddingsEngine.GenerateEmbeddingsFromItem
// It generates embeddings for the given Item using its text representation
func (o *OllamaEmbeddingsEngine) GenerateEmbeddingsFromItem(item types.Item, fields map[uint]types.Facet) (types.Embeddings, error) {
	// Generate a text representation of the item
	itemText := buildItemRepresentation(item, fields)

	// Generate embeddings for the text
	return o.GenerateEmbeddings(itemText)
}

// buildItemRepresentation constructs a string representation of an item
// optimized for generating meaningful embeddings
func buildItemRepresentation(item types.Item, fields map[uint]types.Facet) string {
	var builder strings.Builder

	// Add title with higher weight (repeat twice)
	text, err := item.GetEmbeddingsText()
	if err != nil {
		return item.GetTitle()
	}
	builder.WriteString(text)
	builder.WriteString("\n")

	for fieldId, value := range item.GetFields() {
		f, ok := fields[fieldId]
		if !ok {
			continue
		}
		b := f.GetBaseField()
		if b.HideFacet {
			continue
		}

		if b.Name != "" {
			builder.WriteString(b.Name)
			builder.WriteString(": ")
		}
		builder.WriteString(fmt.Sprintf("%v", value))
		builder.WriteString("\n")
	}
	// Add other string representations
	// strList := item.ToStringList()
	// for _, str := range strList {
	// 	builder.WriteString(str)
	// 	builder.WriteString(" ")
	// }

	// // Add all field values as strings
	// fields := item.GetFields()
	// for _, value := range fields {
	// 	if str, ok := value.(string); ok {
	// 		builder.WriteString(str)
	// 		builder.WriteString(" ")
	// 	} else if str, ok := value.(fmt.Stringer); ok {
	// 		builder.WriteString(str.String())
	// 		builder.WriteString(" ")
	// 	}
	// }

	return builder.String()
}
