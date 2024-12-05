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
	level    int
	id       uint
	Key      string  `json:"key"`
	Value    *string `json:"value"`
	parent   *Category
	Children map[uint]*Category `json:"children"`
}

type Index struct {
	mu         sync.RWMutex
	categories map[uint]*Category
	Facets     map[uint]types.Facet

	DefaultFacets Facets
	Items         map[uint]*types.Item
	ItemsInStock  map[string]types.ItemList

	AutoSuggest   AutoSuggest
	ChangeHandler ChangeHandler
	Sorting       *Sorting
	Search        *search.FreeTextIndex
}

func NewIndex(freeText *search.FreeTextIndex) *Index {
	return &Index{
		mu:           sync.RWMutex{},
		categories:   make(map[uint]*Category),
		Facets:       make(map[uint]types.Facet),
		Items:        make(map[uint]*types.Item),
		ItemsInStock: make(map[string]types.ItemList),
		AutoSuggest:  AutoSuggest{Trie: search.NewTrie()},
		Search:       freeText,
	}
}

func (i *Index) AddKeyField(field *types.BaseField) {
	facet := facet.EmptyKeyValueField(field)
	i.Facets[field.Id] = facet
}

func (i *Index) AddDecimalField(field *types.BaseField) {
	i.Facets[field.Id] = facet.EmptyDecimalField(field)
}

func (i *Index) AddIntegerField(field *types.BaseField) {
	i.Facets[field.Id] = facet.EmptyIntegerField(field)
}

func (i *Index) SetBaseSortMap(sortMap map[uint]float64) {
	if i.Search != nil {
		i.Search.BaseSortMap = sortMap
	}
}

func makeCategoryId(level int, value string) uint {
	return facet.HashString(fmt.Sprintf("%d%s", level, value))
}

func (i *Index) addItemValues(item types.Item) {
	for _, stock := range i.ItemsInStock {
		delete(stock, item.GetId())
	}

	for _, stock := range item.GetStock() {
		stockLocation, ok := i.ItemsInStock[stock.Id]
		if !ok {
			i.ItemsInStock[stock.Id] = types.ItemList{item.GetId(): struct{}{}}
		} else {
			stockLocation[item.GetId()] = struct{}{}
		}
	}

	tree := make([]*Category, 0)
	var base *types.BaseField

	for id, fieldValue := range item.GetFields() {
		if f, ok := i.Facets[id]; ok {
			base = f.GetBaseField()
			if base.CategoryLevel > 0 {
				value, ok := fieldValue.(string)
				if ok {
					cid := makeCategoryId(base.CategoryLevel, value)
					if i.categories[cid] == nil {
						i.categories[cid] = &Category{Value: &value, level: base.CategoryLevel, id: id}
					}
					tree = append(tree, i.categories[cid])
				}
			}
			if !base.HideFacet {

				ok := f.AddValueLink(fieldValue, item)
				if !ok {
					delete(i.Facets, id)
				}
			}

		} else {
			delete(i.Facets, id)
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

func (i *Index) GetCategories() []*Category {
	i.Lock()
	defer i.Unlock()
	categories := make([]*Category, 0)
	for _, category := range i.categories {
		if category.parent == nil && category.level == 1 {
			categories = append(categories, category)
		}
	}
	return categories
}

func (i *Index) removeItemValues(item types.Item) {
	iid := item.GetId()
	for id, fieldValue := range item.GetFields() {
		if f, ok := i.Facets[id]; ok {
			f.RemoveValueLink(fieldValue, iid)
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
	changed := make([]types.Item, 0)
	price_lowered := make([]types.Item, 0)

	for _, it := range items {
		if i.UpsertItemUnsafe(it) {
			price_lowered = append(price_lowered, it)

		}
		changed = append(changed, it)
	}
	if i.ChangeHandler != nil {
		i.ChangeHandler.ItemsUpserted(changed)
		i.ChangeHandler.PriceLowered(price_lowered)
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
	current, isUpdate := i.Items[item.GetId()]
	if item.IsDeleted() {
		if isUpdate {
			i.deleteItemUnsafe(item.GetId())
		}
		return false
	}
	if isUpdate {
		old_price := (*current).GetPrice()
		new_price := item.GetPrice()
		if new_price < old_price {
			price_lowered = true
		}
		i.removeItemValues(*current)
	}
	go noUpdates.Inc()
	//	i.AllItems[item.Id] = &item.ItemFields
	i.addItemValues(item)

	i.Items[item.GetId()] = &item
	if i.ChangeHandler != nil {
		return price_lowered
	}
	go i.AutoSuggest.InsertItem(item)
	if i.Search != nil {
		go i.Search.CreateDocument(item.GetId(), item.ToString())
	}
	if i.Sorting != nil {
		i.Sorting.IndexChanged(i)
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
