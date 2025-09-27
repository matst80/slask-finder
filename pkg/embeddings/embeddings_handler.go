package embeddings

import (
	"iter"
	"log"
	"maps"
	"sync"

	"github.com/matst80/slask-finder/pkg/types"
)

// EmbeddingsRate is the rate limit for embedding requests per second
type EmbeddingsRate = float64

type EmbeddingsClient interface {
	GetEmbeddings(itemId uint) (types.Embeddings, bool)
}

type EmbeddingsClientHandler struct {
	Embeddings       map[uint]types.Embeddings
	EmbeddingsEngine types.EmbeddingsEngine
}

// ItemEmbeddingsHandler handles embeddings-related operations for items
// It implements the types.ItemHandler interface
type ItemEmbeddingsHandler struct {
	mu               sync.RWMutex
	Embeddings       map[uint]types.Embeddings
	EmbeddingsEngine types.EmbeddingsEngine
	EmbeddingsQueue  *EmbeddingsQueue
}

// ItemEmbeddingsHandlerOptions contains configuration options for creating a new embeddings handler
type ItemEmbeddingsHandlerOptions struct {
	EmbeddingsEngine    types.EmbeddingsEngine
	EmbeddingsWorkers   int            // Number of workers in the embeddings queue
	EmbeddingsQueueSize int            // Size of the embeddings queue buffer
	EmbeddingsRateLimit EmbeddingsRate // Rate limit for embedding requests per second
}

// DefaultEmbeddingsHandlerOptions returns default configuration options for embeddings handler creation
func DefaultEmbeddingsHandlerOptions(engine types.EmbeddingsEngine) ItemEmbeddingsHandlerOptions {
	return ItemEmbeddingsHandlerOptions{
		EmbeddingsEngine:    engine,
		EmbeddingsWorkers:   4,       // Default to 4 workers
		EmbeddingsQueueSize: 1000000, // Use a very large queue size (effectively unlimited)
		EmbeddingsRateLimit: 0.0,     // No rate limit
	}
}

// NewItemEmbeddingsHandler creates a new ItemEmbeddingsHandler with the specified options
func NewItemEmbeddingsHandler(opts ItemEmbeddingsHandlerOptions, queueDone func(items map[uint]types.Embeddings) error) *ItemEmbeddingsHandler {
	handler := &ItemEmbeddingsHandler{
		mu:               sync.RWMutex{},
		Embeddings:       make(map[uint]types.Embeddings),
		EmbeddingsEngine: opts.EmbeddingsEngine,
	}

	// Initialize embeddings queue if an embeddings engine is available
	if opts.EmbeddingsEngine != nil {
		// Create a store function that safely stores embeddings in the handler
		storeFunc := func(itemId uint, emb types.Embeddings) {
			handler.mu.Lock()
			defer handler.mu.Unlock()
			handler.Embeddings[itemId] = emb
		}

		// Create the embeddings queue with configured workers and effectively unlimited queue size
		handler.EmbeddingsQueue = NewEmbeddingsQueue(
			opts.EmbeddingsEngine,
			storeFunc,
			func() error {
				handler.mu.RLock()
				defer handler.mu.RUnlock()
				return queueDone(handler.Embeddings)
			},
			opts.EmbeddingsWorkers,
			opts.EmbeddingsQueueSize)

		// Start the queue
		handler.EmbeddingsQueue.Start()

		log.Printf("Initialized embeddings queue with %d workers and unlimited queue size",
			opts.EmbeddingsWorkers)
	}

	return handler
}

// HandleItem implements types.ItemHandler interface
// Processes a single item for embeddings generation
func (h *ItemEmbeddingsHandler) HandleItem(item types.Item) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.handleItemUnsafe(item)
}

// HandleItems implements types.ItemHandler interface
// Processes multiple items for embeddings generation
func (h *ItemEmbeddingsHandler) HandleItems(items iter.Seq[types.Item]) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for item := range items {
		h.handleItemUnsafe(item)
	}
}

// HandleItemUnsafe implements types.ItemHandler interface
// Processes an item for embeddings generation without acquiring locks
func (h *ItemEmbeddingsHandler) handleItemUnsafe(item types.Item) {

	id := item.GetId()

	// Check if we already have embeddings for this item
	_, hasEmbeddings := h.Embeddings[id]

	// Queue item for embeddings generation if needed
	if !hasEmbeddings && h.EmbeddingsQueue != nil && !item.IsDeleted() && item.CanHaveEmbeddings() {
		if !h.EmbeddingsQueue.QueueItem(item) {
			log.Printf("Failed to queue item %d for embeddings generation after timeout", id)
		}
	}
}

// Cleanup stops the embeddings queue and performs any necessary cleanup
func (h *ItemEmbeddingsHandler) Cleanup() {
	if h.EmbeddingsQueue != nil {
		h.EmbeddingsQueue.Stop()
	}
}

// GetEmbeddingsQueueStatus returns the current length and capacity of the embeddings queue
func (h *ItemEmbeddingsHandler) GetEmbeddingsQueueStatus() (queueLength int, queueCapacity int, hasQueue bool) {
	if h.EmbeddingsQueue == nil {
		return 0, 0, false
	}
	return h.EmbeddingsQueue.QueueLength(), h.EmbeddingsQueue.QueueCapacity(), true
}

// GetEmbeddingsQueueDetails returns detailed information about the embeddings queue status
func (h *ItemEmbeddingsHandler) GetEmbeddingsQueueDetails() map[string]interface{} {
	if h.EmbeddingsQueue == nil {
		return map[string]interface{}{
			"hasQueue": false,
		}
	}

	return h.EmbeddingsQueue.Status()
}

// GetEmbeddings returns the embeddings for a specific item ID
func (h *ItemEmbeddingsHandler) GetEmbeddings(itemId uint) (types.Embeddings, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	emb, exists := h.Embeddings[itemId]
	return emb, exists
}

// HasEmbeddings checks if embeddings exist for a specific item ID
func (h *ItemEmbeddingsHandler) HasEmbeddings(itemId uint) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, exists := h.Embeddings[itemId]
	return exists
}

// RemoveEmbeddings removes embeddings for a specific item ID
func (h *ItemEmbeddingsHandler) RemoveEmbeddings(itemId uint) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.Embeddings, itemId)
}

// GetAllEmbeddings returns a copy of all embeddings for persistence operations
func (h *ItemEmbeddingsHandler) GetAllEmbeddings() map[uint]types.Embeddings {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Create a copy to avoid race conditions
	result := make(map[uint]types.Embeddings, len(h.Embeddings))
	for id, emb := range h.Embeddings {
		result[id] = emb
	}
	return result
}

// LoadEmbeddings loads embeddings from a map for initialization
func (h *ItemEmbeddingsHandler) LoadEmbeddings(embeddings map[uint]types.Embeddings) {
	h.mu.Lock()
	defer h.mu.Unlock()
	maps.Copy(h.Embeddings, embeddings)
}

// GetEmbeddingsEngine returns the embeddings engine for external use
func (h *ItemEmbeddingsHandler) GetEmbeddingsEngine() types.EmbeddingsEngine {
	return h.EmbeddingsEngine
}
