package index

import (
	"context"
	"iter"
	"log"
	"sync"

	"git.tornberg.me/mats/go-redis-inventory/pkg/inventory"
	"github.com/RoaringBitmap/roaring/v2"
	"github.com/matst80/slask-finder/pkg/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"
)

var (
	noUpdates = promauto.NewCounter(prometheus.CounterOpts{
		Name: "slaskfinder_index_updates_total",
		Help: "The total number of item updates",
	})
	noDeletes = promauto.NewCounter(prometheus.CounterOpts{
		Name: "slaskfinder_index_deletes_total",
		Help: "The total number of item deletions",
	})
	noInserts = promauto.NewCounter(prometheus.CounterOpts{
		Name: "slaskfinder_index_inserts_total",
		Help: "The total number of item insertions",
	})
)

// stockEntry is defined in stock_entry.go:
// It internally maintains a roaring bitmap (lock-free copy-on-write).
// Helpers: add(id uint), remove(id uint), bitmap() *roaring.Bitmap

// ItemIndexWithStock maintains item and stock indexes using only sync.Map
// for concurrency (no outer RWMutex layer).
// - Items:       sync.Map mapping uint -> types.Item
// - ItemsBySku:  sync.Map mapping string -> uint
// - ItemsInStock: sync.Map mapping string -> *stockEntry (set of item IDs)
type ItemIndexWithStock struct {
	Items        sync.Map // map[uint]types.Item
	ItemsBySku   sync.Map // map[string]uint
	ItemsInStock sync.Map // map[string]*stockEntry
}

func NewIndexWithStock() *ItemIndexWithStock {
	return &ItemIndexWithStock{}
}

// addItemValues indexes the item in stock locations.
func (i *ItemIndexWithStock) addItemValues(item types.Item) {
	itemId := item.GetId()
	for stockLocId := range item.GetStock() {
		entryAny, loaded := i.ItemsInStock.Load(stockLocId)
		var entry *stockEntry
		if !loaded {
			entry = newStockEntry()
			actual, double := i.ItemsInStock.LoadOrStore(stockLocId, entry)
			if double {
				entry = actual.(*stockEntry)
			}
		} else {
			entry = entryAny.(*stockEntry)
		}
		entry.add(uint32(itemId))
	}
}

// removeItemValues removes an item id from all stock entries.
func (i *ItemIndexWithStock) removeItemValues(item types.Item) {
	itemId := item.GetId()
	i.ItemsInStock.Range(func(_, value any) bool {
		entry := value.(*stockEntry)
		entry.remove(uint32(itemId))
		return true
	})
}

// HandleItem processes a single item asynchronously using wg.Go.
func (i *ItemIndexWithStock) HandleItem(item types.Item, wg *sync.WaitGroup) {
	wg.Go(func() {
		i.handleItemUnsafe(item)
	})
}

func (i *ItemIndexWithStock) HandleStockUpdate(changes []inventory.InventoryChange) {
	for _, change := range changes {
		log.Printf("got stock update: %v", change)
		v, ok := i.ItemsBySku.Load(change.SKU)
		if ok {
			item, found := i.GetItem(v.(types.ItemId))
			if found {
				newValue := uint16(change.Value)
				item.UpdateStock(change.StockLocationID, newValue)

				if newValue > 0 {
					// Add to stock location
					entryAny, loaded := i.ItemsInStock.Load(change.StockLocationID)
					var entry *stockEntry
					if !loaded {
						entry = newStockEntry()
						actual, double := i.ItemsInStock.LoadOrStore(change.StockLocationID, entry)
						if double {
							entry = actual.(*stockEntry)
						}
					} else {
						entry = entryAny.(*stockEntry)
					}
					entry.add(uint32(item.GetId()))
				} else if newValue == 0 {
					// Remove from stock location
					if entryAny, ok := i.ItemsInStock.Load(change.StockLocationID); ok {
						entry := entryAny.(*stockEntry)
						entry.remove(uint32(item.GetId()))
					}
				}

			}
		}
	}
}

// HandleItems processes a sequence of items without a global lock.
func (i *ItemIndexWithStock) HandleItems(it iter.Seq[types.Item]) {
	for item := range it {
		i.handleItemUnsafe(item)
	}
}

// handleItemUnsafe performs the mutation; concurrency safety relies on sync.Map and per-stockEntry locks.
func (i *ItemIndexWithStock) handleItemUnsafe(item types.Item) {
	id := item.GetId()

	existingAny, isUpdate := i.Items.Load(id)
	if isUpdate {
		if existing, ok := existingAny.(types.Item); ok {
			i.removeItemValues(existing)
		}
		if item.IsDeleted() {
			i.Items.Delete(id)
			i.ItemsBySku.Delete(item.GetSku())
			noDeletes.Inc()
			return
		}
	}

	if item.IsDeleted() {
		return
	}
	if !isUpdate {
		noInserts.Inc()
	}

	i.Items.Store(id, item)
	i.ItemsBySku.Store(item.GetSku(), id)
	i.addItemValues(item)
	noUpdates.Inc()
}

// GetStockResult merges item sets for provided stock location IDs without
// materializing intermediate ItemLists, aggregating roaring bitmaps directly.
func (i *ItemIndexWithStock) GetStockResult(stockLocations []string) *types.ItemList {
	if len(stockLocations) == 0 {
		return &types.ItemList{}
	}
	acc := roaring.NewBitmap()
	for _, stockId := range stockLocations {
		if entryAny, ok := i.ItemsInStock.Load(stockId); ok {
			acc.Or(entryAny.(*stockEntry).bitmap())
		}
	}
	return types.FromBitmap(acc)
}

var (
	name   = "slask-finder-index"
	tracer = otel.Tracer(name)
)

func SpannedFetcher(fn func() *types.ItemList, name string) func(ctx context.Context) *types.ItemList {
	return func(ctx context.Context) *types.ItemList {
		// Here you could add tracing or logging using the name parameter.
		_, span := tracer.Start(ctx, name)
		defer span.End()
		return fn()
	}
}

// MatchStock integrates stock filtering into a QueryMerger.
func (i *ItemIndexWithStock) MatchStock(stockLocations []string, qm *types.QueryMerger) {
	if len(stockLocations) > 0 {
		qm.Add(SpannedFetcher(func() *types.ItemList {
			return i.GetStockResult(stockLocations)
		}, "MatchStock"))
	}
}

// GetAllItems returns a sequence iterating over all items.
func (i *ItemIndexWithStock) GetAllItems() iter.Seq[types.Item] {
	return func(yield func(types.Item) bool) {
		// Direct iteration over sync.Map.
		i.Items.Range(func(_, value any) bool {
			return yield(value.(types.Item))
		})
	}
}

// GetItemBySku retrieves an item by SKU.
func (i *ItemIndexWithStock) GetItemBySku(sku string) (types.Item, bool) {
	if idAny, ok := i.ItemsBySku.Load(sku); ok {
		id := idAny.(types.ItemId)
		return i.GetItem(id)
	}
	return nil, false
}

// GetItem retrieves an item by id.
func (i *ItemIndexWithStock) GetItem(id types.ItemId) (types.Item, bool) {
	if val, ok := i.Items.Load(id); ok {
		return val.(types.Item), true
	}
	return nil, false
}

// HasItem checks if an item exists.
func (i *ItemIndexWithStock) HasItem(id uint) bool {
	_, ok := i.Items.Load(id)
	return ok
}

// GetItems returns a sequence for a set of ids.
func (i *ItemIndexWithStock) GetItems(ids iter.Seq[types.ItemId]) iter.Seq[types.Item] {
	return func(yield func(types.Item) bool) {
		for id := range ids {
			if item, ok := i.GetItem(id); ok {
				if !yield(item) {
					return
				}
			}
		}
	}
}
