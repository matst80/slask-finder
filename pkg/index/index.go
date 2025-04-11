package index

import (
	"cmp"
	"fmt"
	"log"
	"slices"
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
}

type UpdateHandler interface {
	UpsertItems(item []types.Item)
	DeleteItem(id uint)
}

type Category struct {
	level int
	id    uint
	//Key      string  `json:"key"`
	Value    *string `json:"value"`
	parent   *Category
	Children map[uint]*Category `json:"children"`
}

type Index struct {
	mu sync.RWMutex
	//categories    map[uint]*Category
	Facets        map[uint]types.Facet
	ItemFieldIds  map[uint]map[uint]struct{}
	Items         map[uint]*types.Item
	ItemsInStock  map[string]types.ItemList
	IsMaster      bool
	All           types.ItemList
	AutoSuggest   *AutoSuggest
	ChangeHandler ChangeHandler
	Sorting       *Sorting
	Search        *search.FreeTextIndex
}

func NewIndex() *Index {
	return &Index{
		mu:  sync.RWMutex{},
		All: types.ItemList{},
		//categories:   make(map[uint]*Category),
		ItemFieldIds: make(map[uint]map[uint]struct{}),
		Facets:       make(map[uint]types.Facet),
		Items:        make(map[uint]*types.Item),
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

// func (i *Index) SetBaseSortMap(sortMap map[uint]float64) {
// 	if i.Search != nil {
// 		i.Search.BaseSortMap = sortMap
// 	}
// }

func makeCategoryId(level int, value string) uint {
	return facet.HashString(fmt.Sprintf("%d%s", level, value))
}

func (i *Index) addItemValues(item types.Item) {
	if i.IsMaster {
		return
	}
	itemId := item.GetId()
	for _, stock := range i.ItemsInStock {
		delete(stock, itemId)
	}

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

	tree := make([]*Category, 0)
	var base *types.BaseField
	// test virtual category

	for id, fieldValue := range item.GetFields() {
		if f, ok := i.Facets[id]; ok {
			base = f.GetBaseField()
			// if base.CategoryLevel > 0 {
			// 	value, ok := fieldValue.(string)
			// 	if ok {
			// 		cid := makeCategoryId(base.CategoryLevel, value)
			// 		if i.categories[cid] == nil {
			// 			i.categories[cid] = &Category{Value: &value, level: base.CategoryLevel, id: id}
			// 		}
			// 		tree = append(tree, i.categories[cid])
			// 	}
			// }

			if f.AddValueLink(fieldValue, item) && i.ItemFieldIds != nil && !base.HideFacet {
				if fids, ok := i.ItemFieldIds[itemId]; ok {
					fids[base.Id] = struct{}{}
				} else {
					log.Printf("No field for item id: %d", itemId)
				}
			}

		} else {
			//delete(i.Facets, id)
		}
	}

	if len(tree) > 0 {
		slices.SortFunc(tree, func(a, b *Category) int {
			return cmp.Compare(a.level, b.level)
		})
		for i := 0; i < len(tree)-1; i++ {

			if tree[i].Children == nil {
				tree[i].Children = make(map[uint]*Category, 0)
			}
			id := makeCategoryId(tree[i+1].level, *tree[i+1].Value)
			tree[i].Children[id] = tree[i+1]
			tree[i+1].parent = tree[i]
		}
	}
}

// func (i *Index) GetCategories() []*Category {
// 	i.Lock()
// 	defer i.Unlock()
// 	categories := make([]*Category, 0)
// 	for _, category := range i.categories {
// 		if category.parent == nil && category.level == 1 {
// 			categories = append(categories, category)
// 		}
// 	}
// 	return categories
// }

func (i *Index) removeItemValues(item types.Item) {
	if i.IsMaster {
		return
	}
	itemId := item.GetId()
	for fieldId, fieldValue := range item.GetFields() {
		if f, ok := i.Facets[fieldId]; ok {
			f.RemoveValueLink(fieldValue, itemId)
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
			if (*item).MergeKeyFields(updates) {
				i.UpsertItemUnsafe(*item)
				items = append(items, *item)
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
	//changed := make([]types.Item, 0, len(items))
	//price_lowered := make([]types.Item, 0, len(items))

	for _, it := range items {
		i.UpsertItemUnsafe(it)
		//	price_lowered = append(price_lowered, it)
		//}
		//changed = append(changed, it)
	}
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

func (i *Index) UpsertItemUnsafe(item types.Item) bool {
	price_lowered := false
	id := item.GetId()
	current, isUpdate := i.Items[id]
	if item.IsDeleted() {
		if item.IsSoftDeleted() {
			if isUpdate {
				i.removeItemValues(*current)
			}
			return false
		}
		delete(i.ItemFieldIds, id)
		if isUpdate {
			i.deleteItemUnsafe(id)
		}
		return false
	}
	i.All.AddId(id)
	if isUpdate {
		old_price := (*current).GetPrice()
		new_price := item.GetPrice()
		if new_price < old_price {
			price_lowered = true
		}
		i.removeItemValues(*current)
	}
	go noUpdates.Inc()
	i.ItemFieldIds[id] = make(map[uint]struct{})
	//	i.AllItems[item.Id] = &item.ItemFields
	if i.Search != nil {
		i.addItemValues(item)
	}

	i.Items[id] = &item
	if i.IsMaster {
		return price_lowered
	}

	if i.AutoSuggest != nil {
		go i.AutoSuggest.InsertItem(item)
	}
	if i.Search != nil {
		go i.Search.CreateDocument(id, item.ToStringList()...)
	}
	return price_lowered
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
		i.removeItemValues(*item)
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

func (i *Index) GetItemIds(ids []uint, page int, pageSize int) []uint {
	l := len(ids)
	start := page * pageSize
	end := min(l, start+pageSize)
	if start > l {
		return ids[0:0]
	}
	return ids[start:end]
}
