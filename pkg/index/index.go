package index

import (
	"cmp"
	"fmt"
	"log"
	"slices"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"tornberg.me/facet-search/pkg/facet"
	"tornberg.me/facet-search/pkg/search"
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

type KeyFacet = facet.KeyField
type DecimalFacet = facet.NumberField[float64]
type IntFacet = facet.NumberField[int]

type ChangeHandler interface {
	//ItemChanged(item *DataItem)
	ItemDeleted(id uint)
	ItemsUpserted(item []DataItem)
	PriceLowered(item []DataItem)
}

type UpdateHandler interface {
	UpsertItems(item []DataItem)
	DeleteItem(id uint)
}

type Category struct {
	level    int
	id       uint
	Value    string `json:"value"`
	parent   *Category
	Children map[string]*Category `json:"children"`
}

type Index struct {
	mu            sync.RWMutex
	categories    map[string]*Category
	KeyFacets     map[uint]*KeyFacet
	DecimalFacets map[uint]*DecimalFacet
	IntFacets     map[uint]*IntFacet
	DefaultFacets Facets
	Items         map[uint]*DataItem
	ItemsInStock  map[string]facet.IdList
	//AllItems      facet.MatchList
	AutoSuggest   AutoSuggest
	ChangeHandler ChangeHandler
	Sorting       *Sorting
	Search        *search.FreeTextIndex
}

func NewIndex(freeText *search.FreeTextIndex) *Index {
	return &Index{
		mu:            sync.RWMutex{},
		categories:    make(map[string]*Category),
		KeyFacets:     make(map[uint]*KeyFacet),
		DecimalFacets: make(map[uint]*DecimalFacet),
		IntFacets:     make(map[uint]*IntFacet),
		Items:         make(map[uint]*DataItem),
		ItemsInStock:  make(map[string]facet.IdList),
		// AllItems:      facet.MatchList{},
		AutoSuggest: AutoSuggest{Trie: search.NewTrie()},
		Search:      freeText,
	}
}

// func (i *Index) CreateDefaultFacets(sort *facet.SortIndex) {
// 	ids := facet.IdList{}
// 	for id := range i.AllItems {
// 		ids[id] = struct{}{}
// 	}
// 	i.DefaultFacets = i.GetFacetsFromResult(&ids, &Filters{}, sort)
// }

func (i *Index) AddKeyField(field *facet.BaseField) {
	i.KeyFacets[field.Id] = facet.EmptyKeyValueField(field)
}

func (i *Index) AddDecimalField(field *facet.BaseField) {
	i.DecimalFacets[field.Id] = facet.EmptyNumberField[float64](field)
}

func (i *Index) AddIntegerField(field *facet.BaseField) {
	i.IntFacets[field.Id] = facet.EmptyNumberField[int](field)
}

func (i *Index) SetBaseSortMap(sortMap map[uint]float64) {
	if i.Search != nil {
		i.Search.BaseSortMap = sortMap
	}
}

func (i *Index) addItemValues(item *DataItem) {
	for _, stock := range i.ItemsInStock {
		delete(stock, item.Id)
	}
	if item.Stock != nil {
		for _, stock := range item.Stock {
			stockLocation, ok := i.ItemsInStock[stock.Id]
			if !ok {
				i.ItemsInStock[stock.Id] = facet.IdList{item.Id: struct{}{}}
			} else {
				stockLocation[item.Id] = struct{}{}
			}
		}
	}
	tree := make([]*Category, 0)
	if item.Fields != nil {
		for _, field := range item.Fields {
			if field.Value == "" || len(field.Value) > 64 {
				continue
			}

			if f, ok := i.KeyFacets[field.Id]; ok {
				if f.CategoryLevel > 0 {
					id := fmt.Sprintf("%d%s", f.CategoryLevel, field.Value)
					if i.categories[id] == nil {
						i.categories[id] = &Category{Value: field.Value, level: f.CategoryLevel, id: f.Id}
					}
					tree = append(tree, i.categories[id])
				}

				f.AddValueLink(field.Value, item.Id)

			}
		}
	}
	if len(tree) > 0 {
		slices.SortFunc(tree, func(a, b *Category) int {
			return cmp.Compare(a.level, b.level)
		})
		for i := 0; i < len(tree)-1; i++ {

			if tree[i].Children == nil {
				tree[i].Children = make(map[string]*Category, 0)
			}
			id := fmt.Sprintf("%d%s", tree[i+1].level, tree[i+1].Value)
			tree[i].Children[id] = tree[i+1]
			tree[i+1].parent = tree[i]
		}
	}
	if item.DecimalFields != nil {
		for _, field := range item.DecimalFields {
			if field.Value == 0.0 {
				continue
			}
			if f, ok := i.DecimalFacets[field.Id]; ok {
				f.AddValueLink(field.Value, item.Id)
			}
		}
	}
	if item.IntegerFields != nil {

		for _, field := range item.IntegerFields {
			if field.Value == 0 {
				continue
			}
			if f, ok := i.IntFacets[field.Id]; ok {
				f.AddValueLink(field.Value, item.Id)
			}
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

func (i *Index) removeItemValues(item *DataItem) {
	for _, field := range item.Fields {
		if f, ok := i.KeyFacets[field.Id]; ok {
			f.RemoveValueLink(field.Value, item.Id)
		}
	}
	for _, field := range item.DecimalFields {
		if f, ok := i.DecimalFacets[field.Id]; ok {
			f.RemoveValueLink(field.Value, item.Id)
		}
	}
	for _, field := range item.IntegerFields {
		if f, ok := i.IntFacets[field.Id]; ok {
			f.RemoveValueLink(field.Value, item.Id)
		}
	}
}

func (i *Index) UpsertItem(item *DataItem) {
	log.Printf("Upserting item %d", item.Id)
	i.mu.Lock()
	defer i.mu.Unlock()
	i.UpsertItemUnsafe(item)
}

type CategoryUpdate struct {
	Id    uint   `json:"id"`
	Value string `json:"value"`
}

func (i *Index) UpdateCategoryValues(ids []uint, updates []CategoryUpdate) {
	i.mu.Lock()
	defer i.mu.Unlock()
	items := make([]DataItem, 0)
	for _, id := range ids {
		item, ok := i.Items[id]
		if ok {
			item.MergeKeyFields(updates)
			i.UpsertItemUnsafe(item)
			items = append(items, *item)
		}
	}
	if i.ChangeHandler != nil {
		i.ChangeHandler.ItemsUpserted(items)
	}
}

func (i *Index) UpsertItems(items []DataItem) {
	l := len(items)
	if l == 0 {
		return
	}
	log.Printf("Upserting items %d", l)
	i.mu.Lock()
	defer i.mu.Unlock()
	price_lowered := make([]DataItem, l)
	j := 0
	for _, it := range items {
		if i.UpsertItemUnsafe(&it) {
			price_lowered[j] = it
			j++
		}
	}
	if i.ChangeHandler != nil {
		i.ChangeHandler.ItemsUpserted(items)
		i.ChangeHandler.PriceLowered(price_lowered[0:j])
	}
}

func (i *Index) Lock() {
	i.mu.RLock()
}

func (i *Index) Unlock() {
	i.mu.RUnlock()
}

func (i *Index) UpsertItemUnsafe(item *DataItem) bool {
	price_lowered := false
	current, isUpdate := i.Items[item.Id]
	if item.SaleStatus == "MDD" {
		if isUpdate {
			i.deleteItemUnsafe(item.Id)
		}
		return price_lowered
	}
	if isUpdate {
		old_price := current.GetPrice()
		new_price := item.GetPrice()
		if new_price < old_price {
			price_lowered = true
		}
		i.removeItemValues(current)
	}
	go noUpdates.Inc()
	//	i.AllItems[item.Id] = &item.ItemFields
	i.addItemValues(item)

	i.Items[item.Id] = item
	if i.ChangeHandler != nil {
		return price_lowered
	}
	go i.AutoSuggest.InsertItem(item)
	if i.Search != nil {
		go i.Search.CreateDocument(item.Id, item.Sku, item.Title, item.BulletPoints)
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

func (i *Index) GetItemIds(ids []uint, page int, pageSize int) []uint {
	l := len(ids)
	start := page * pageSize
	end := min(l, start+pageSize)
	if start > l {
		return ids[0:0]
	}
	return ids[start:end]
}

func (i *Index) GetItems(ids []uint, page int, pageSize int) []ResultItem {
	items := make([]ResultItem, min(len(ids), pageSize))
	idx := 0
	for _, id := range i.GetItemIds(ids, page, pageSize) {
		item, ok := i.Items[id]
		if ok {
			items[idx] = MakeResultItem(item)
			idx++
		}
	}
	return items[0:idx]
}
