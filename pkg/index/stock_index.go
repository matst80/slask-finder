package index

import (
	"log"

	"github.com/matst80/slask-finder/pkg/types"
)

type ItemIndexWithStock struct {
	*ItemIndex
	ItemsBySku   map[string]types.Item
	ItemsInStock map[string]types.ItemList
	All          types.ItemList
}

func NewIndexWithStock() *ItemIndexWithStock {
	idx := &ItemIndexWithStock{
		ItemIndex:    NewItemIndex(),
		All:          types.ItemList{},
		ItemsBySku:   make(map[string]types.Item),
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
	log.Printf("Handling item %d", item.GetId())
	i.mu.Lock()
	defer i.mu.Unlock()
	i.HandleItemUnsafe(item)
}

func (i *ItemIndexWithStock) HandleItems(items []types.Item) {
	l := len(items)
	if l == 0 {
		return
	}
	log.Printf("Handling items %d", l)
	i.mu.Lock()
	defer i.mu.Unlock()

	for _, it := range items {
		i.HandleItemUnsafe(it)
	}
}

func (i *ItemIndexWithStock) Lock() {
	i.mu.RLock()
}

func (i *ItemIndexWithStock) Unlock() {
	i.mu.RUnlock()
}

func (i *ItemIndexWithStock) HandleItemUnsafe(item types.Item) {

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
	i.ItemsBySku[item.GetSku()] = item

	i.addItemValues(item)
}

func (i *ItemIndexWithStock) HasItem(id uint) bool {
	_, ok := i.Items[id]
	return ok
}
