package server

import (
	"log"

	"github.com/matst80/slask-finder/pkg/index"
	"github.com/matst80/slask-finder/pkg/search"
	"github.com/matst80/slask-finder/pkg/types"
)

// InitializeHandlers sets up all the item handlers for the WebServer
// This should be called after the Index is created but before processing items
func (ws *WebServer) InitializeHandlers(opts HandlerOptions) error {
	log.Println("Initializing WebServer handlers...")

	// Initialize facet handler
	facetOpts := index.FacetItemHandlerOptions{}
	ws.FacetHandler = index.NewFacetItemHandler(facetOpts)

	// Initialize embeddings handler if an embeddings engine is available
	if opts.EmbeddingsEngine != nil {
		embeddingsOpts := index.ItemEmbeddingsHandlerOptions{
			EmbeddingsEngine:    opts.EmbeddingsEngine,
			EmbeddingsWorkers:   opts.EmbeddingsWorkers,
			EmbeddingsQueueSize: opts.EmbeddingsQueueSize,
			EmbeddingsRateLimit: opts.EmbeddingsRateLimit,
		}

		ws.EmbeddingsHandler = index.NewItemEmbeddingsHandler(embeddingsOpts, opts.QueueDoneCallback)
	}

	// Initialize search handler if search is enabled
	if opts.EnableSearch {
		searchOpts := index.FreeTextItemHandlerOptions{
			Tokenizer: &search.Tokenizer{MaxTokens: opts.SearchMaxTokens},
		}
		ws.SearchHandler = index.NewFreeTextItemHandler(searchOpts)
	}

	// Initialize sorting handler
	sortingOpts := index.SortingItemHandlerOptions{
		RedisAddr:     opts.RedisAddr,
		RedisPassword: opts.RedisPassword,
		RedisDB:       opts.RedisDB,
	}
	ws.SortingHandler = index.NewSortingItemHandler(sortingOpts)

	// Keep backward compatibility with existing Sorting field
	if ws.SortingHandler != nil && ws.SortingHandler.Sorting != nil {
		ws.Sorting = ws.SortingHandler.Sorting
	}

	log.Println("WebServer handlers initialized successfully")
	return nil
}

// HandlerOptions contains configuration for all handlers
type HandlerOptions struct {
	// Embeddings configuration
	EmbeddingsEngine    types.EmbeddingsEngine
	EmbeddingsWorkers   int
	EmbeddingsQueueSize int
	EmbeddingsRateLimit index.EmbeddingsRate
	QueueDoneCallback   func() error

	// Search configuration
	EnableSearch    bool
	SearchMaxTokens int

	// Sorting configuration
	RedisAddr     string
	RedisPassword string
	RedisDB       int
}

// DefaultHandlerOptions returns default configuration for handlers
func DefaultHandlerOptions(engine types.EmbeddingsEngine, queueDone func() error) HandlerOptions {
	return HandlerOptions{
		EmbeddingsEngine:    engine,
		EmbeddingsWorkers:   4,
		EmbeddingsQueueSize: 1000000,
		EmbeddingsRateLimit: 10.0,
		QueueDoneCallback:   queueDone,
		EnableSearch:        true,
		SearchMaxTokens:     128,
	}
}

// UpsertItems coordinates the upsert operation across all handlers
func (ws *WebServer) UpsertItems(items []types.Item) {
	// Update the core index first
	ws.Index.HandleItems(items)

	// Then update all handlers
	if ws.FacetHandler != nil {
		ws.FacetHandler.HandleItems(items)
	}

	if ws.SearchHandler != nil {
		ws.SearchHandler.Lock()
		defer ws.SearchHandler.Unlock()

		for _, item := range items {
			itemStringList := item.ToStringList()
			ws.SearchHandler.CreateDocumentUnsafe(item.GetId(), itemStringList...)
		}
	}

	if ws.EmbeddingsHandler != nil {
		for _, item := range items {
			if !item.IsSoftDeleted() && item.CanHaveEmbeddings() {
				if err := ws.EmbeddingsHandler.HandleItemUnsafe(item); err != nil {
					log.Printf("Error handling embeddings for item %d: %v", item.GetId(), err)
				}
			}
		}
	}

	if ws.SortingHandler != nil {
		ws.SortingHandler.IndexChanged(ws.Index)
	}
}
