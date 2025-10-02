package index

import (
	"iter"
	"sync"

	"github.com/matst80/slask-finder/pkg/types"
)

type ItemIndexWithStock struct {
	*ItemIndex
	ItemsBySku   map[string]uint
	ItemsInStock map[string]types.ItemList
}

func NewIndexWithStock() *ItemIndexWithStock {
	idx := &ItemIndexWithStock{
		ItemIndex:    NewItemIndex(),
		ItemsBySku:   make(map[string]uint),
		ItemsInStock: make(map[string]types.ItemList),
	}

	return idx
}

func (i *ItemIndexWithStock) addItemValues(item types.Item) {
	itemId := item.GetId()

	for id, _ := range item.GetStock() {
		// if stock == "" || stock == "0" {
		// 	continue
		// }
		stockLocation, ok := i.ItemsInStock[id]
		if !ok {
			i.ItemsInStock[id] = types.ItemList{itemId: struct{}{}}
		} else {
			stockLocation[itemId] = struct{}{}
		}
	}
}

func (i *ItemIndexWithStock) removeItemValues(item types.Item) {

	itemId := item.GetId()
	for _, stock := range i.ItemsInStock {
		delete(stock, itemId)
	}
}

func (i *ItemIndexWithStock) HandleItem(item types.Item, wg *sync.WaitGroup) {
	wg.Go(func() {
		i.mu.Lock()
		defer i.mu.Unlock()
		i.handleItemUnsafe(item)
	})
}

func (i *ItemIndexWithStock) HandleItems(it iter.Seq[types.Item]) {
	i.mu.Lock()
	defer i.mu.Unlock()
	for item := range it {
		i.handleItemUnsafe(item)
	}
}

func (i *ItemIndexWithStock) handleItemUnsafe(item types.Item) {

	i.ItemIndex.handleItemUnsafe(item)

	id := item.GetId()
	current, isUpdate := i.Items[id]
	if isUpdate {
		i.removeItemValues(current)
	}
	if item.IsDeleted() {
		delete(i.ItemsBySku, item.GetSku())
		return
	}

	i.ItemsBySku[item.GetSku()] = id

	i.addItemValues(item)
}

func (i *ItemIndexWithStock) GetStockResult(stockLocations []string) *types.ItemList {
	resultStockIds := &types.ItemList{}
	for _, stockId := range stockLocations {
		stockIds, ok := i.ItemsInStock[stockId]
		if ok {
			resultStockIds.Merge(&stockIds)
		}
	}
	return resultStockIds
}

func (i *ItemIndexWithStock) MatchStock(stockLocations []string, qm *types.QueryMerger) {
	if len(stockLocations) > 0 {
		qm.Add(func() *types.ItemList {
			return i.GetStockResult(stockLocations)
		})
	}
}

func (i *ItemIndexWithStock) GetItemBySku(sku string) (types.Item, bool) {
	if id, ok := i.ItemsBySku[sku]; ok {
		return i.GetItem(id)
	}
	return nil, false
}

func (i *ItemIndexWithStock) GetItems(ids iter.Seq[uint]) iter.Seq[types.Item] {
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
