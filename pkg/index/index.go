package index

import (
	"log"
	"sync"

	"github.com/matst80/slask-finder/pkg/embeddings"
	"github.com/matst80/slask-finder/pkg/facet"
	"github.com/matst80/slask-finder/pkg/search"
	"github.com/matst80/slask-finder/pkg/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// EmbeddingsRate is the rate limit for embedding requests per second
type EmbeddingsRate = float64

var (
	noUpdates = promauto.NewCounter(prometheus.CounterOpts{
		Name: "slaskfinder_index_updates_total",
		Help: "The total number of item updates",
	})
	noDeletes = promauto.NewCounter(prometheus.CounterOpts{
		Name: "slaskfinder_index_deletes_total",
		Help: "The total number of item deletions",
	})
)

type ChangeHandler interface {
	//ItemChanged(item *DataItem)
	ItemDeleted(id uint)
	ItemsUpserted(item []types.Item)
	PriceLowered(item []types.Item)
	FieldsChanged(item []types.FieldChange)
}

type UpdateHandler interface {
	UpsertItems(item []types.Item)
	UpdateFields(changes []types.FieldChange)
	DeleteItem(id uint)
}

// type Category struct {
// 	level int
// 	id    uint
// 	//Key      string  `json:"key"`
// 	Value    *string `json:"value"`
// 	parent   *Category
// 	Children map[uint]*Category `json:"children"`
// }

type Index struct {
	mu           sync.RWMutex
	EmbeddingsMu sync.RWMutex
	//categories    map[uint]*Category
	Facets       map[uint]types.Facet
	ItemFieldIds map[uint]types.ItemList
	Items        map[uint]types.Item
	ItemsBySku   map[string]*types.Item
	ItemsInStock map[string]types.ItemList
	Embeddings   map[uint]types.Embeddings
	IsMaster     bool
	All          types.ItemList
	//AutoSuggest   *AutoSuggest
	ChangeHandler    ChangeHandler
	Sorting          *Sorting
	Search           *search.FreeTextIndex
	EmbeddingsEngine types.EmbeddingsEngine
	EmbeddingsQueue  *embeddings.EmbeddingsQueue
}

// IndexOptions contains configuration options for creating a new index
type IndexOptions struct {
	EmbeddingsEngine    types.EmbeddingsEngine
	EmbeddingsWorkers   int            // Number of workers in the embeddings queue
	EmbeddingsQueueSize int            // Size of the embeddings queue buffer
	EmbeddingsRateLimit EmbeddingsRate // Rate limit for embedding requests per second
}

// DefaultIndexOptions returns default configuration options for index creation
func DefaultIndexOptions(engine types.EmbeddingsEngine) IndexOptions {
	return IndexOptions{
		EmbeddingsEngine:    engine,
		EmbeddingsWorkers:   4,       // Default to 2 workers
		EmbeddingsQueueSize: 1000000, // Use a very large queue size (effectively unlimited)
		EmbeddingsRateLimit: 0.0,     // No rate limit
	}
}

func NewIndex(engine types.EmbeddingsEngine) *Index {
	opts := DefaultIndexOptions(engine)
	return NewIndexWithOptions(opts)
}

func NewIndexWithOptions(opts IndexOptions) *Index {
	idx := &Index{
		mu:           sync.RWMutex{},
		EmbeddingsMu: sync.RWMutex{},
		All:          types.ItemList{},
		//categories:   make(map[uint]*Category),
		Embeddings:       make(map[uint]types.Embeddings),
		ItemsBySku:       make(map[string]*types.Item),
		ItemFieldIds:     make(map[uint]types.ItemList),
		Facets:           make(map[uint]types.Facet),
		Items:            make(map[uint]types.Item),
		ItemsInStock:     make(map[string]types.ItemList),
		EmbeddingsEngine: opts.EmbeddingsEngine,
	}

	// Initialize embeddings queue if an embeddings engine is available
	if opts.EmbeddingsEngine != nil {
		// Create a store function that safely stores embeddings in the index
		storeFunc := func(itemId uint, emb types.Embeddings) {
			idx.EmbeddingsMu.Lock()
			defer idx.EmbeddingsMu.Unlock()
			idx.Embeddings[itemId] = emb

		}

		// Create the embeddings queue with configured workers and effectively unlimited queue size
		idx.EmbeddingsQueue = embeddings.NewEmbeddingsQueue(
			opts.EmbeddingsEngine,
			idx.Facets,
			storeFunc,
			opts.EmbeddingsWorkers,
			opts.EmbeddingsQueueSize)

		// Start the queue
		idx.EmbeddingsQueue.Start()

		log.Printf("Initialized embeddings queue with %d workers and unlimited queue size",
			opts.EmbeddingsWorkers)
	}

	return idx
}

func (i *Index) AddKeyField(field *types.BaseField) {
	i.Facets[field.Id] = facet.EmptyKeyValueField(field)
}

func (i *Index) AddDecimalField(field *types.BaseField) {
	i.Facets[field.Id] = facet.EmptyDecimalField(field)
}

func (i *Index) AddIntegerField(field *types.BaseField) {
	i.Facets[field.Id] = facet.EmptyIntegerField(field)
}

func (i *Index) GetKeyFacet(id uint) (*facet.KeyField, bool) {
	if f, ok := i.Facets[id]; ok {
		switch tf := f.(type) {
		case facet.KeyField:
			return &tf, true
		case *facet.KeyField:
			return tf, true
		}
	}
	return nil, false
}

func (i *Index) addItemValues(item types.Item) {

	itemId := item.GetId()

	for id, stock := range item.GetStock() {
		if stock == "" || stock == "0" {
			continue
		}
		stockLocation, ok := i.ItemsInStock[id]
		if !ok {
			i.ItemsInStock[id] = types.ItemList{itemId: struct{}{}}
		} else {
			stockLocation[itemId] = struct{}{}
		}
	}

	for id, fieldValue := range item.GetFields() {
		if f, ok := i.Facets[id]; ok {
			if !f.IsExcludedFromFacets() && f.AddValueLink(fieldValue, itemId) && i.ItemFieldIds != nil {
				if fids, ok := i.ItemFieldIds[itemId]; ok {
					fids.AddId(id)
				} else {
					log.Printf("No field for item id: %d, id: %d", itemId, id)
				}
			}

		}
	}
}

func (i *Index) removeItemValues(item types.Item) {
	if i.IsMaster {
		return
	}

	itemId := item.GetId()
	for _, stock := range i.ItemsInStock {
		delete(stock, itemId)
	}
	for fieldId, fieldValue := range item.GetFields() {
		if f, ok := i.Facets[fieldId]; ok {
			f.RemoveValueLink(fieldValue, itemId)
		}
	}

}

func (i *Index) UpdateFields(changes []types.FieldChange) {
	i.mu.Lock()
	defer i.mu.Unlock()
	log.Printf("Updating fields %d", len(changes))
	for _, change := range changes {
		if change.Action == types.ADD_FIELD {
			log.Println("not implemented add field")
		} else {
			if f, ok := i.Facets[change.Id]; ok {
				switch change.Action {
				case types.UPDATE_FIELD:
					f.UpdateBaseField(change.BaseField)

				case types.REMOVE_FIELD:
					delete(i.Facets, change.Id)
				}
			}
		}
	}

}

func (i *Index) UpsertItem(item types.Item) {
	log.Printf("Upserting item %d", item.GetId())
	i.mu.Lock()
	defer i.mu.Unlock()
	i.UpsertItemUnsafe(item)

	// // Handle embeddings for individual items
	// if !hasEmbeddings && i.EmbeddingsQueue != nil && !item.IsSoftDeleted() && item.CanHaveEmbeddings() {
	// 	// Use the enhanced QueueItem with timeout
	// 	if !i.EmbeddingsQueue.QueueItem(item) {
	// 		log.Printf("Failed to queue item %d for embeddings generation after timeout", item.GetId())
	// 	}
	// }
}

func (i *Index) UpdateCategoryValues(ids []uint, updates []types.CategoryUpdate) {
	i.mu.Lock()
	defer i.mu.Unlock()
	items := make([]types.Item, 0)
	for _, id := range ids {
		item, ok := i.Items[id]
		if ok {
			if item.MergeKeyFields(updates) {
				i.UpsertItemUnsafe(item)
				items = append(items, item)
			}
		}
	}
	if i.ChangeHandler != nil {
		i.ChangeHandler.ItemsUpserted(items)
	}
}

func (i *Index) UpsertItems(items []types.Item) {
	l := len(items)
	if l == 0 {
		return
	}
	log.Printf("Upserting items %d", l)
	i.mu.Lock()
	defer i.mu.Unlock()
	// if i.AutoSuggest != nil {
	// 	i.AutoSuggest.Lock()
	// 	defer i.AutoSuggest.Unlock()
	// }
	if i.Search != nil {
		i.Search.Lock()
		defer i.Search.Unlock()
	}

	// Collect items that need embeddings

	for _, it := range items {
		i.UpsertItemUnsafe(it)

	}

	go noUpdates.Add(float64(l))
	if i.ChangeHandler != nil {
		log.Printf("Propagating changes")
		go i.ChangeHandler.ItemsUpserted(items)
	}

	if i.Sorting != nil {
		i.Sorting.IndexChanged(i)
	}

}

func (i *Index) Lock() {
	i.mu.RLock()
}

func (i *Index) Unlock() {
	i.mu.RUnlock()
}

func (i *Index) UpsertItemUnsafe(item types.Item) {
	//price_lowered := false
	id := item.GetId()
	current, isUpdate := i.Items[id]
	if item.IsDeleted() {
		delete(i.All, id)
		delete(i.ItemsBySku, item.GetSku())
		delete(i.ItemFieldIds, id)
		if item.IsSoftDeleted() {
			if isUpdate {
				i.removeItemValues(current)
			}
			return
		}

		if isUpdate {
			i.deleteItemUnsafe(id)
		}

		// nothing more to do when item is deleted
		return
	}

	if isUpdate {
		// should probably be a merge here instead
		i.removeItemValues(current)
	}

	i.Items[id] = item
	if i.IsMaster {
		// Master index does not maintain facets or search index, but extracts embeddings for storage
		i.EmbeddingsMu.RLock()
		_, hasEmbeddings := i.Embeddings[id]
		i.EmbeddingsMu.RUnlock()
		if !hasEmbeddings && i.EmbeddingsQueue != nil && !item.IsSoftDeleted() && item.CanHaveEmbeddings() {
			i.EmbeddingsQueue.QueueItem(item)
		}

		return
	} else {
		i.ItemFieldIds[id] = make(types.ItemList, len(item.GetFields()))
		i.All.AddId(id)
		i.ItemsBySku[item.GetSku()] = &item

		i.addItemValues(item)

		item.UpdateBasePopularity(*types.CurrentSettings.PopularityRules)

		if i.Search != nil {
			i.Search.CreateDocumentUnsafe(id, item.ToStringList()...)
		}
	}
}

func (i *Index) DeleteItem(id uint) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.deleteItemUnsafe(id)
}

func (i *Index) deleteItemUnsafe(id uint) {
	item, ok := i.Items[id]
	if ok {
		noDeletes.Inc()
		i.removeItemValues(item)
		delete(i.Items, id)
		// delete(i.AllItems, id)
		if i.ChangeHandler != nil {
			i.ChangeHandler.ItemDeleted(id)
		}
	}
}

func (i *Index) HasItem(id uint) bool {
	_, ok := i.Items[id]
	return ok
}

// func (i *Index) GetItemIds(ids []uint, page int, pageSize int) []uint {
// 	l := len(ids)
// 	start := page * pageSize
// 	end := min(l, start+pageSize)
// 	if start > l {
// 		return ids[0:0]
// 	}
// 	return ids[start:end]
// }

// GenerateAndStoreEmbeddings generates embeddings for an item and stores them in the index
// This is kept for backwards compatibility but the preferred method is using EmbeddingsQueue
func (i *Index) GenerateAndStoreEmbeddings(item types.Item) error {
	if i.EmbeddingsEngine == nil {
		// Skip if no embeddings engine is available
		return nil
	}

	id := item.GetId()

	// Generate embeddings for the item
	embeddings, err := i.EmbeddingsEngine.GenerateEmbeddingsFromItem(item, i.Facets)
	if err != nil {
		return err
	}

	// Safely store the embeddings
	i.mu.Lock()
	i.Embeddings[id] = embeddings
	i.mu.Unlock()

	return nil
}

// Cleanup stops the embeddings queue and performs any necessary cleanup
func (i *Index) Cleanup() {
	if i.EmbeddingsQueue != nil {
		i.EmbeddingsQueue.Stop()
	}
}

// GetEmbeddingsQueueStatus returns the current length and capacity of the embeddings queue
func (i *Index) GetEmbeddingsQueueStatus() (queueLength int, queueCapacity int, hasQueue bool) {
	if i.EmbeddingsQueue == nil {
		return 0, 0, false
	}
	return i.EmbeddingsQueue.QueueLength(), i.EmbeddingsQueue.QueueCapacity(), true
}

// GetEmbeddingsQueueDetails returns detailed information about the embeddings queue status
func (i *Index) GetEmbeddingsQueueDetails() map[string]interface{} {
	if i.EmbeddingsQueue == nil {
		return map[string]interface{}{
			"hasQueue": false,
		}
	}

	return i.EmbeddingsQueue.Status()
}
