package index

import (
	"iter"

	"github.com/matst80/slask-finder/pkg/types"
)

type ItemIndexWithStock struct {
	*ItemIndex
	ItemsBySku   map[string]uint
	ItemsInStock map[string]types.ItemList
	All          types.ItemList
}

func NewIndexWithStock() *ItemIndexWithStock {
	idx := &ItemIndexWithStock{
		ItemIndex:    NewItemIndex(),
		All:          types.ItemList{},
		ItemsBySku:   make(map[string]uint),
		ItemsInStock: make(map[string]types.ItemList),
	}

	return idx
}

func (i *ItemIndexWithStock) addItemValues(item types.Item) {
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

func (i *ItemIndexWithStock) removeItemValues(item types.Item) {

	itemId := item.GetId()
	for _, stock := range i.ItemsInStock {
		delete(stock, itemId)
	}
}

func (i *ItemIndexWithStock) HandleItem(item types.Item) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.handleItemUnsafe(item)
}

func (i *ItemIndexWithStock) HandleItems(it iter.Seq[types.Item]) {
	i.mu.Lock()
	defer i.mu.Unlock()
	for item := range it {
		i.handleItemUnsafe(item)
	}
}

func (i *ItemIndexWithStock) handleItemUnsafe(item types.Item) {
	i.ItemIndex.handleItem(item)
	i.mu.Lock()
	defer i.mu.Unlock()
	id := item.GetId()
	current, isUpdate := i.Items[id]
	if isUpdate {
		i.removeItemValues(current)
		delete(i.Items, id)
	}
	if item.IsDeleted() {
		delete(i.All, id)
		delete(i.ItemsBySku, item.GetSku())
	}

	i.Items[id] = item

	i.All.AddId(id)
	i.ItemsBySku[item.GetSku()] = id

	i.addItemValues(item)
}

func (i *ItemIndexWithStock) HasItem(id uint) bool {
	_, ok := i.Items[id]
	return ok
}
