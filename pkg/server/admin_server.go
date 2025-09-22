package server

import (
	"net/http"

	"github.com/matst80/slask-finder/pkg/index"
	"github.com/matst80/slask-finder/pkg/search"
	"github.com/matst80/slask-finder/pkg/storage"
	"github.com/matst80/slask-finder/pkg/tracking"
	"github.com/matst80/slask-finder/pkg/types"
	"golang.org/x/oauth2"
)

type BaseWebServer struct {
	OAuthConfig *oauth2.Config
}

type WebServerMiddleware interface {
	Handle() *http.ServeMux
}

// AdminWebServer contains all handlers needed for administrative operations
// This includes full facet management, search indexing, embeddings processing, and sorting
type AdminWebServer struct {
	*BaseWebServer
	// Core data
	Index        *index.ItemIndex
	Db           *storage.DataRepository
	ContentIndex *index.ContentIndex

	// All handlers - no nil checks needed in admin operations
	FacetHandler      *index.FacetItemHandler
	SearchHandler     *index.FreeTextItemHandler
	EmbeddingsHandler *index.ItemEmbeddingsHandler
	SortingHandler    *index.SortingItemHandler

	// Additional services
	Cache            *Cache
	Sorting          *index.Sorting // Keep for backward compatibility
	Tracking         tracking.Tracking
	FacetLimit       int
	SearchFacetLimit int
	FieldData        map[string]*FieldData
	PriceWatches     *PriceWatchesData
}

// NewAdminWebServer creates a new admin server with all handlers initialized
func NewAdminWebServer(idx *index.ItemIndex, db *storage.DataRepository, contentIdx *index.ContentIndex) *AdminWebServer {
	return &AdminWebServer{
		Index:            idx,
		Db:               db,
		ContentIndex:     contentIdx,
		FacetLimit:       1024,
		SearchFacetLimit: 10280,
		FieldData:        map[string]*FieldData{},
	}
}

// InitializeHandlers sets up all the item handlers for the AdminWebServer
func (aws *AdminWebServer) InitializeHandlers(opts HandlerOptions) error {
	// Initialize facet handler - always present in admin
	facetOpts := index.FacetItemHandlerOptions{}
	aws.FacetHandler = index.NewFacetItemHandler(facetOpts)

	// Initialize search handler - always present in admin
	searchOpts := index.FreeTextItemHandlerOptions{
		Tokenizer: &search.Tokenizer{MaxTokens: opts.SearchMaxTokens},
	}
	aws.SearchHandler = index.NewFreeTextItemHandler(searchOpts)

	// Initialize embeddings handler - always present in admin
	if opts.EmbeddingsEngine != nil {
		embeddingsOpts := index.ItemEmbeddingsHandlerOptions{
			EmbeddingsEngine:    opts.EmbeddingsEngine,
			EmbeddingsWorkers:   opts.EmbeddingsWorkers,
			EmbeddingsQueueSize: opts.EmbeddingsQueueSize,
			EmbeddingsRateLimit: opts.EmbeddingsRateLimit,
		}
		aws.EmbeddingsHandler = index.NewItemEmbeddingsHandler(embeddingsOpts, opts.QueueDoneCallback)
	}

	// Initialize sorting handler - always present in admin
	sortingOpts := index.SortingItemHandlerOptions{
		RedisAddr:     opts.RedisAddr,
		RedisPassword: opts.RedisPassword,
		RedisDB:       opts.RedisDB,
	}
	aws.SortingHandler = index.NewSortingItemHandler(sortingOpts)

	// Keep backward compatibility
	if aws.SortingHandler != nil && aws.SortingHandler.Sorting != nil {
		aws.Sorting = aws.SortingHandler.Sorting
	}

	return nil
}

// UpsertItems coordinates the upsert operation across all admin handlers
func (aws *AdminWebServer) UpsertItems(items []types.Item) {
	// Update the core index first

	aws.Index.HandleItems(items)

	// Update all handlers - no nil checks needed in admin server
	aws.FacetHandler.HandleItems(items)

	aws.SearchHandler.Lock()
	defer aws.SearchHandler.Unlock()

	for _, item := range items {
		itemStringList := item.ToStringList()
		aws.SearchHandler.CreateDocumentUnsafe(item.GetId(), itemStringList...)
	}

	for _, item := range items {
		if !item.IsSoftDeleted() && item.CanHaveEmbeddings() {
			if err := aws.EmbeddingsHandler.HandleItemUnsafe(item); err != nil {
				// log.Printf("Error handling embeddings for item %d: %v", item.GetId(), err)
			}
		}
	}

	aws.SortingHandler.IndexChanged(aws.Index)
}

// GetFacet returns a facet by ID - no nil check needed
func (aws *AdminWebServer) GetFacet(id uint) (types.Facet, bool) {
	facet, ok := aws.FacetHandler.Facets[id]
	return facet, ok
}

// GetAllFacets returns all facets - no nil check needed
func (aws *AdminWebServer) GetAllFacets() map[uint]types.Facet {
	return aws.FacetHandler.Facets
}

// SearchItems performs search - no nil check needed
func (aws *AdminWebServer) SearchItems(query string) *types.ItemList {
	return aws.SearchHandler.Search(query)
}

// FilterItems filters items - no nil check needed
func (aws *AdminWebServer) FilterItems(query string, res *types.ItemList) {
	aws.SearchHandler.Filter(query, res)
}

// GetEmbeddings returns embeddings for an item - no nil check needed
func (aws *AdminWebServer) GetEmbeddings(itemId uint) (types.Embeddings, bool) {
	return aws.EmbeddingsHandler.GetEmbeddings(itemId)
}

// HasEmbeddings checks if embeddings exist - no nil check needed
func (aws *AdminWebServer) HasEmbeddings(itemId uint) bool {
	return aws.EmbeddingsHandler.HasEmbeddings(itemId)
}
