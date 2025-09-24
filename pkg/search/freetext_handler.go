package search

import (
	"sync"

	"github.com/matst80/slask-finder/pkg/types"
)

// FreeTextItemHandler handles free text search operations for items
// It implements the types.ItemHandler interface
type FreeTextItemHandler struct {
	mu    sync.RWMutex
	Index *FreeTextIndex
}

// FreeTextItemHandlerOptions contains configuration options for creating a new free text handler
type FreeTextItemHandlerOptions struct {
	Tokenizer *Tokenizer
}

// DefaultFreeTextHandlerOptions returns default configuration options for free text handler creation
func DefaultFreeTextHandlerOptions() FreeTextItemHandlerOptions {
	return FreeTextItemHandlerOptions{
		Tokenizer: &Tokenizer{MaxTokens: 128},
	}
}

// NewFreeTextItemHandler creates a new FreeTextItemHandler with the specified options
func NewFreeTextItemHandler(opts FreeTextItemHandlerOptions) *FreeTextItemHandler {
	handler := &FreeTextItemHandler{
		mu:    sync.RWMutex{},
		Index: NewFreeTextIndex(opts.Tokenizer),
	}

	return handler
}

// HandleItem implements types.ItemHandler interface
// Processes a single item for free text search indexing
func (h *FreeTextItemHandler) HandleItem(item types.Item) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.HandleItemUnsafe(item)
}

// HandleItems implements types.ItemHandler interface
// Processes multiple items for free text search indexing
func (h *FreeTextItemHandler) HandleItems(items []types.Item) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, item := range items {
		h.HandleItemUnsafe(item)
	}

}

// HandleItemUnsafe implements types.ItemHandler interface
// Processes an item for free text search indexing without acquiring locks
func (h *FreeTextItemHandler) HandleItemUnsafe(item types.Item) {
	if item == nil {
		return
	}

	id := item.GetId()

	// Only create search documents for items that are not deleted or soft deleted
	if !item.IsDeleted() && !item.IsSoftDeleted() {
		// Create search document using item's string representation
		h.Index.CreateDocumentUnsafe(id, item.ToStringList()...)
	}

}

// Lock implements types.ItemHandler interface
func (h *FreeTextItemHandler) Lock() {
	h.mu.RLock()
}

// Unlock implements types.ItemHandler interface
func (h *FreeTextItemHandler) Unlock() {
	h.mu.RUnlock()
}

// Search performs a free text search and returns matching item IDs
func (h *FreeTextItemHandler) Search(query string) *types.ItemList {
	if h.Index == nil {
		return &types.ItemList{}
	}
	return h.Index.Search(query)
}

// Filter filters the provided item list based on the query
func (h *FreeTextItemHandler) Filter(query string, res *types.ItemList) {
	if h.Index != nil {
		h.Index.Filter(query, res)
	}
}

// CreateDocument creates a search document for the given item ID and text
func (h *FreeTextItemHandler) CreateDocument(id uint, text ...string) {
	if h.Index != nil {
		h.Index.CreateDocument(id, text...)
	}
}

// CreateDocumentUnsafe creates a search document without acquiring locks
func (h *FreeTextItemHandler) CreateDocumentUnsafe(id uint, text ...string) {
	if h.Index != nil {
		h.Index.CreateDocumentUnsafe(id, text...)
	}
}

// RemoveDocument removes a document from the search index
func (h *FreeTextItemHandler) RemoveDocument(id uint, text ...string) {
	if h.Index != nil {
		h.Index.RemoveDocument(id, text...)
	}
}

// GetFreeTextIndex returns the underlying FreeTextIndex for external access
func (h *FreeTextItemHandler) GetFreeTextIndex() *FreeTextIndex {
	return h.Index
}

// FindTrieMatchesForWord finds trie matches for a single word
func (h *FreeTextItemHandler) FindTrieMatchesForWord(word string, resultChan chan<- []Match) {
	if h.Index != nil {
		h.Index.FindTrieMatchesForWord(word, resultChan)
	} else {
		resultChan <- []Match{}
	}
}

// FindTrieMatchesForContext finds trie matches with context
func (h *FreeTextItemHandler) FindTrieMatchesForContext(prevWord string, word string, resultChan chan<- []Match) {
	if h.Index != nil {
		h.Index.FindTrieMatchesForContext(prevWord, word, resultChan)
	} else {
		resultChan <- []Match{}
	}
}

// GetTrie returns the underlying Trie for external access
func (h *FreeTextItemHandler) GetTrie() *Trie {
	if h.Index == nil {
		return nil
	}
	return h.Index.Trie
}
