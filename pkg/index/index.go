package index

import (
	"log"
	"sync"

	"github.com/matst80/slask-finder/pkg/facet"
	"github.com/matst80/slask-finder/pkg/search"
	"github.com/matst80/slask-finder/pkg/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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
	mu sync.RWMutex
	//categories    map[uint]*Category
	Facets       map[uint]types.Facet
	ItemFieldIds map[uint]types.ItemList
	Items        map[uint]types.Item
	ItemsBySku   map[string]*types.Item
	ItemsInStock map[string]types.ItemList
	IsMaster     bool
	All          types.ItemList
	//AutoSuggest   *AutoSuggest
	ChangeHandler ChangeHandler
	Sorting       *Sorting
	Search        *search.FreeTextIndex
}

func NewIndex() *Index {
	return &Index{
		mu:  sync.RWMutex{},
		All: types.ItemList{},
		//categories:   make(map[uint]*Category),
		ItemsBySku:   make(map[string]*types.Item),
		ItemFieldIds: make(map[uint]types.ItemList),
		Facets:       make(map[uint]types.Facet),
		Items:        make(map[uint]types.Item),
		ItemsInStock: make(map[string]types.ItemList),
	}
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

// func (i *Index) SetBaseSortMap(sortMap map[uint]float64) {
// 	if i.Search != nil {
// 		i.Search.BaseSortMap = sortMap
// 	}
// }

// func makeCategoryId(level int, value string) uint {
// 	return facet.HashString(fmt.Sprintf("%d%s", level, value))
// }

func (i *Index) addItemValues(item types.Item) {
	if i.IsMaster {
		return
	}
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
			if f.AddValueLink(fieldValue, itemId) && i.ItemFieldIds != nil && !f.IsExcludedFromFacets() {
				if fids, ok := i.ItemFieldIds[itemId]; ok {
					fids[id] = struct{}{}
				} else {
					log.Printf("No field for item id: %d", itemId)
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
				if change.Action == types.UPDATE_FIELD {
					f.UpdateBaseField(change.BaseField)

				} else if change.Action == types.REMOVE_FIELD {
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

	//changed := make([]types.Item, 0, len(items))
	//price_lowered := make([]types.Item, 0, len(items))

	for _, it := range items {
		i.UpsertItemUnsafe(it)
		//	price_lowered = append(price_lowered, it)
		//}
		//changed = append(changed, it)
	}
	go noUpdates.Add(float64(l))
	if i.ChangeHandler != nil {
		log.Printf("Propagating changes")
		go i.ChangeHandler.ItemsUpserted(items)
		//i.ChangeHandler.PriceLowered(price_lowered)
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
		return
	}

	if isUpdate {
		// old_price := (*current).GetPrice()
		// new_price := item.GetPrice()
		// if new_price < old_price {
		// 	price_lowered = true
		// }
		i.removeItemValues(current)
	}

	i.Items[id] = item
	if i.IsMaster {
		return
	} else {
		i.ItemFieldIds[id] = make(types.ItemList, len(item.GetFields()))
		i.All.AddId(id)
		i.ItemsBySku[item.GetSku()] = &item
		if i.Search != nil {
			i.addItemValues(item)
		}

		item.UpdateBasePopularity(*types.CurrentSettings.PopularityRules)
		// if i.AutoSuggest != nil {
		// 	i.AutoSuggest.InsertItemUnsafe(item)
		// }
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
