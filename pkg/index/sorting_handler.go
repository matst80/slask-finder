package index

import (
	"sync"

	"github.com/matst80/slask-finder/pkg/types"
)

type SortingItemHandler struct {
	mu      sync.RWMutex
	Sorting *Sorting
}

type SortingItemHandlerOptions struct {
	RedisAddr     string
	RedisPassword string
	RedisDB       int
}

func NewSortingItemHandler(opts SortingItemHandlerOptions) *SortingItemHandler {
	handler := &SortingItemHandler{}

	// Initialize sorting if Redis config provided
	if opts.RedisAddr != "" {
		handler.Sorting = NewSorting(opts.RedisAddr, opts.RedisPassword, opts.RedisDB)
	}

	return handler
}

// ItemHandler interface implementation
func (h *SortingItemHandler) HandleItem(item types.Item) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.HandleItemUnsafe(item)
}

func (h *SortingItemHandler) HandleItems(items []types.Item) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, item := range items {
		h.HandleItemUnsafe(item)
	}
}

func (h *SortingItemHandler) HandleItemUnsafe(item types.Item) {
	// Sorting handler doesn't need to process individual items during upsert
	// Sorting happens on demand or periodically
}

func (h *SortingItemHandler) Lock() {
	h.mu.Lock()
}

func (h *SortingItemHandler) Unlock() {
	h.mu.Unlock()
}

// Sorting-specific methods
func (h *SortingItemHandler) InitializeWithIndex(idx *ItemIndex, facetIndex *FacetItemHandler) {
	if h.Sorting != nil {
		h.Sorting.InitializeWithIndex(idx, facetIndex)
	}
}

func (h *SortingItemHandler) StartListeningForChanges() {
	if h.Sorting != nil {
		h.Sorting.StartListeningForChanges()
	}
}

func (h *SortingItemHandler) IndexChanged(idx *ItemIndex) {
	if h.Sorting != nil {
		h.Sorting.IndexChanged(idx)
	}
}

func (h *SortingItemHandler) Close() error {
	if h.Sorting != nil {
		return h.Sorting.Close()
	}
	return nil
}

// Delegation methods for backward compatibility
func (h *SortingItemHandler) GetSort(id string) *types.ByValue {
	if h.Sorting != nil {
		return h.Sorting.GetSort(id)
	}
	return nil
}

func (h *SortingItemHandler) GetSortedFields(items []*JsonFacet) []*JsonFacet {
	if h.Sorting != nil {
		return h.Sorting.GetSortedFields(items)
	}
	return items
}
