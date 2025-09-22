package server

import (
	"github.com/matst80/slask-finder/pkg/index"
	"github.com/matst80/slask-finder/pkg/search"
	"github.com/matst80/slask-finder/pkg/storage"
	"github.com/matst80/slask-finder/pkg/tracking"
	"github.com/matst80/slask-finder/pkg/types"
)

// ClientWebServer contains minimal handlers needed for client operations
// This avoids unnecessary handlers and nil checks for client-only functionality
type ClientWebServer struct {
	*BaseWebServer
	// Core data - always present
	Index            *index.ItemIndex
	Db               *storage.DataRepository
	ContentIndex     *index.ContentIndex
	EmbeddingsClient index.EmbeddingsClient

	// Minimal handlers for client operations
	FacetHandler  *index.FacetItemHandler    // For reading facets only
	SearchHandler *index.FreeTextItemHandler // For search functionality

	// Optional handlers - may be nil in client mode
	SortingHandler *index.SortingItemHandler // Only if Redis is available

	// Additional services
	Cache            *Cache
	Sorting          *index.Sorting // Keep for backward compatibility
	Tracking         tracking.Tracking
	FacetLimit       int
	SearchFacetLimit int
	FieldData        map[string]*FieldData
	PriceWatches     *PriceWatchesData
}

// NewClientWebServer creates a new client server with minimal handlers
func NewClientWebServer(idx *index.ItemIndex, db *storage.DataRepository, contentIdx *index.ContentIndex) *ClientWebServer {
	return &ClientWebServer{
		Index:            idx,
		Db:               db,
		ContentIndex:     contentIdx,
		FacetLimit:       1024,
		SearchFacetLimit: 10280,
		FieldData:        map[string]*FieldData{},
	}
}

// InitializeClientHandlers sets up only the handlers needed for client operations
func (cws *ClientWebServer) InitializeClientHandlers(opts ClientHandlerOptions) error {
	// Initialize facet handler for reading facets
	facetOpts := index.FacetItemHandlerOptions{}
	cws.FacetHandler = index.NewFacetItemHandler(facetOpts)

	// Initialize search handler if needed
	if opts.EnableSearch {
		searchOpts := index.FreeTextItemHandlerOptions{
			Tokenizer: &search.Tokenizer{MaxTokens: opts.SearchMaxTokens},
		}
		cws.SearchHandler = index.NewFreeTextItemHandler(searchOpts)
	}

	// Initialize sorting handler only if Redis is available
	if opts.RedisAddr != "" {
		sortingOpts := index.SortingItemHandlerOptions{
			RedisAddr:     opts.RedisAddr,
			RedisPassword: opts.RedisPassword,
			RedisDB:       opts.RedisDB,
		}
		cws.SortingHandler = index.NewSortingItemHandler(sortingOpts)

		// Keep backward compatibility
		if cws.SortingHandler != nil && cws.SortingHandler.Sorting != nil {
			cws.Sorting = cws.SortingHandler.Sorting
		}
	}

	return nil
}

// ClientHandlerOptions contains configuration for client handlers
type ClientHandlerOptions struct {
	// Search configuration
	EnableSearch    bool
	SearchMaxTokens int

	// Sorting configuration (optional)
	RedisAddr     string
	RedisPassword string
	RedisDB       int
}

// DefaultClientHandlerOptions returns default configuration for client handlers
func DefaultClientHandlerOptions() ClientHandlerOptions {
	return ClientHandlerOptions{
		EnableSearch:    true,
		SearchMaxTokens: 128,
	}
}

// GetFacet returns a facet by ID - no nil check needed for FacetHandler
func (cws *ClientWebServer) GetFacet(id uint) (types.Facet, bool) {
	facet, ok := cws.FacetHandler.Facets[id]
	return facet, ok
}

// GetAllFacets returns all facets - no nil check needed for FacetHandler
func (cws *ClientWebServer) GetAllFacets() map[uint]types.Facet {
	return cws.FacetHandler.Facets
}

// SearchItems performs search if SearchHandler is available
func (cws *ClientWebServer) SearchItems(query string) *types.ItemList {
	if cws.SearchHandler != nil {
		return cws.SearchHandler.Search(query)
	}
	return &types.ItemList{}
}

// FilterItems filters items if SearchHandler is available
func (cws *ClientWebServer) FilterItems(query string, res *types.ItemList) {
	if cws.SearchHandler != nil {
		cws.SearchHandler.Filter(query, res)
	}
}

// GetEmbeddings is not available in client mode
func (cws *ClientWebServer) GetEmbeddings(itemId uint) (types.Embeddings, bool) {
	return nil, false
}

// HasEmbeddings always returns false in client mode
func (cws *ClientWebServer) HasEmbeddings(itemId uint) bool {
	return false
}
