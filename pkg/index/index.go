package index

import (
	"log"
	"sync"

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

type ItemChangeHandler interface {
	ItemsUpserted(item []types.Item)
}

type ItemUpdateHandler interface {
	UpsertItems(item []types.Item)
}

type FieldChangeHandler interface {
	FieldsChanged(item []types.FieldChange)
}

type UpdateHandler interface {
	UpdateFields(changes []types.FieldChange)
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
	mu sync.RWMutex
	//categories    map[uint]*Category
	Items        map[uint]types.Item
	ItemsBySku   map[string]*types.Item
	ItemsInStock map[string]types.ItemList
	IsMaster     bool
	All          types.ItemList
	//AutoSuggest   *AutoSuggest
	ChangeHandler ItemChangeHandler
}

// // IndexOptions contains configuration options for creating a new index
// type IndexOptions struct {
// 	EmbeddingsEngine    types.EmbeddingsEngine
// 	EmbeddingsWorkers   int            // Number of workers in the embeddings queue
// 	EmbeddingsQueueSize int            // Size of the embeddings queue buffer
// 	EmbeddingsRateLimit EmbeddingsRate // Rate limit for embedding requests per second
// 	EnableSearch        bool           // Enable free text search functionality
// 	SearchMaxTokens     int            // Maximum tokens for search tokenizer
// }

// // DefaultIndexOptions returns default configuration options for index creation
// func DefaultIndexOptions(engine types.EmbeddingsEngine) IndexOptions {
// 	return IndexOptions{
// 		EmbeddingsEngine:    engine,
// 		EmbeddingsWorkers:   4,       // Default to 4 workers
// 		EmbeddingsQueueSize: 1000000, // Use a very large queue size (effectively unlimited)
// 		EmbeddingsRateLimit: 0.0,     // No rate limit
// 		EnableSearch:        true,    // Enable search by default
// 		SearchMaxTokens:     128,     // Default max tokens
// 	}
// }

// func NewIndex(engine types.EmbeddingsEngine, queueDone func(idx *Index) error) *Index {
// 	opts := DefaultIndexOptions(engine)
// 	return NewIndexWithOptions(opts, queueDone)
// }

// NewSimpleIndex creates a basic index without handlers - handlers are managed separately
func NewIndex() *Index {
	idx := &Index{
		mu:  sync.RWMutex{},
		All: types.ItemList{},
		//categories:   make(map[uint]*Category),
		ItemsBySku:   make(map[string]*types.Item),
		Items:        make(map[uint]types.Item, 350_000),
		ItemsInStock: make(map[string]types.ItemList),
	}

	return idx
}

// func NewIndexWithOptions(opts IndexOptions, queueDone func(idx *Index) error) *Index {
// 	idx := &Index{
// 		mu:  sync.RWMutex{},
// 		All: types.ItemList{},
// 		//categories:   make(map[uint]*Category),
// 		ItemsBySku:   make(map[string]*types.Item),
// 		Items:        make(map[uint]types.Item, 100000),
// 		ItemsInStock: make(map[string]types.ItemList),
// 	}

// 	return idx
// }

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
}

func (i *Index) removeItemValues(item types.Item) {
	if i.IsMaster {
		return
	}

	itemId := item.GetId()
	for _, stock := range i.ItemsInStock {
		delete(stock, itemId)
	}
}

func (i *Index) HandleItem(item types.Item) error {
	log.Printf("Handling item %d", item.GetId())
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.HandleItemUnsafe(item)
}

// func (i *Index) UpdateCategoryValues(ids []uint, updates []types.CategoryUpdate) {
// 	i.mu.Lock()
// 	defer i.mu.Unlock()
// 	items := make([]types.Item, 0)
// 	for _, id := range ids {
// 		item, ok := i.Items[id]
// 		if ok {
// 			if item.MergeKeyFields(updates) {
// 				i.UpsertItemUnsafe(item)
// 				items = append(items, item)
// 			}
// 		}
// 	}
// 	if i.ChangeHandler != nil {
// 		i.ChangeHandler.ItemsUpserted(items)
// 	}
// }

func (i *Index) HandleItems(items []types.Item) error {
	l := len(items)
	if l == 0 {
		return nil
	}
	log.Printf("Handling items %d", l)
	i.mu.Lock()
	defer i.mu.Unlock()
	// if i.AutoSuggest != nil {
	// 	i.AutoSuggest.Lock()
	// 	defer i.AutoSuggest.Unlock()
	// }

	// Collect items that need embeddings

	for _, it := range items {
		i.HandleItemUnsafe(it)

	}

	go noUpdates.Add(float64(l))
	if i.ChangeHandler != nil {
		log.Printf("Propagating changes")
		go i.ChangeHandler.ItemsUpserted(items)
	}
	return nil
}

func (i *Index) Lock() {
	i.mu.RLock()
}

func (i *Index) Unlock() {
	i.mu.RUnlock()
}

func (i *Index) HandleItemUnsafe(item types.Item) error {
	//price_lowered := false
	id := item.GetId()
	current, isUpdate := i.Items[id]
	if item.IsDeleted() {
		delete(i.All, id)
		delete(i.ItemsBySku, item.GetSku())
		if item.IsSoftDeleted() {
			if isUpdate {
				i.removeItemValues(current)
			}
			return nil
		}

		if isUpdate {
			i.deleteItemUnsafe(id)
		}

		// nothing more to do when item is deleted
		return nil
	}

	if isUpdate {
		// should probably be a merge here instead
		i.removeItemValues(current)
	}

	i.Items[id] = item
	if i.IsMaster {
		// Master index does not maintain facets or search index
		return nil
	} else {
		i.All.AddId(id)
		i.ItemsBySku[item.GetSku()] = &item

		i.addItemValues(item)

		item.UpdateBasePopularity(*types.CurrentSettings.PopularityRules)
	}
	return nil
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
		// if i.ChangeHandler != nil {
		// 	i.ChangeHandler.ItemDeleted(id)
		// }
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
