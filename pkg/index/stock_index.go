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

func (i *ItemIndexWithStock) HandleItem(item types.Item) error {
	log.Printf("Handling item %d", item.GetId())
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.HandleItemUnsafe(item)
}

func (i *ItemIndexWithStock) HandleItems(items []types.Item) error {
	l := len(items)
	if l == 0 {
		return nil
	}
	log.Printf("Handling items %d", l)
	i.mu.Lock()
	defer i.mu.Unlock()

	for _, it := range items {
		i.HandleItemUnsafe(it)
	}

	return nil
}

func (i *ItemIndexWithStock) Lock() {
	i.mu.RLock()
}

func (i *ItemIndexWithStock) Unlock() {
	i.mu.RUnlock()
}

func (i *ItemIndexWithStock) HandleItemUnsafe(item types.Item) error {

	id := item.GetId()
	current, isUpdate := i.Items[id]
	if isUpdate {
		i.removeItemValues(current)
		delete(i.Items, id)
	}
	if item.IsDeleted() {
		delete(i.All, id)
		delete(i.ItemsBySku, item.GetSku())

		go noDeletes.Inc()
		// nothing more to do when item is deleted
		return nil
	}

	i.Items[id] = item

	i.All.AddId(id)
	i.ItemsBySku[item.GetSku()] = item

	i.addItemValues(item)
	return nil
}

func (i *ItemIndexWithStock) HasItem(id uint) bool {
	_, ok := i.Items[id]
	return ok
}
