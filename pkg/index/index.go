package index

import (
	"sync"

	"tornberg.me/facet-search/pkg/facet"
	"tornberg.me/facet-search/pkg/search"
)

type KeyFacet = facet.KeyField
type DecimalFacet = facet.NumberField[float64]
type IntFacet = facet.NumberField[int]

type ChangeHandler interface {
	ItemChanged(item *DataItem)
	ItemDeleted(id uint)
	ItemAdded(item *DataItem)
}

type UpdateHandler interface {
	UpsertItem(item *DataItem)
	DeleteItem(id uint)
}

type Index struct {
	mu            sync.Mutex
	KeyFacets     map[uint]*KeyFacet
	DecimalFacets map[uint]*DecimalFacet
	IntFacets     map[uint]*IntFacet
	DefaultFacets Facets
	Items         map[uint]*DataItem
	AllItems      facet.MatchList
	AutoSuggest   AutoSuggest
	ChangeHandler ChangeHandler
	Search        *search.FreeTextIndex
}

func NewIndex(freeText *search.FreeTextIndex) *Index {
	return &Index{
		KeyFacets:     make(map[uint]*KeyFacet),
		DecimalFacets: make(map[uint]*DecimalFacet),
		IntFacets:     make(map[uint]*IntFacet),
		Items:         make(map[uint]*DataItem),
		AllItems:      facet.MatchList{},
		AutoSuggest:   AutoSuggest{Trie: search.NewTrie()},
		Search:        freeText,
	}
}

func (i *Index) CreateDefaultFacets(sort *facet.SortIndex) {
	ids := facet.IdList{}
	for id := range i.AllItems {
		ids[id] = struct{}{}
	}
	i.DefaultFacets = i.GetFacetsFromResult(&ids, &Filters{}, sort)
}

func (i *Index) AddKeyField(field *facet.BaseField) {
	i.KeyFacets[field.Id] = facet.EmptyKeyValueField(field)
}

func (i *Index) AddDecimalField(field *facet.BaseField) {
	i.DecimalFacets[field.Id] = facet.EmptyNumberField[float64](field)
}

func (i *Index) AddIntegerField(field *facet.BaseField) {
	i.IntFacets[field.Id] = facet.EmptyNumberField[int](field)
}

func (i *Index) addItemValues(item *DataItem) {

	if item.Fields != nil {
		for _, field := range item.Fields {
			if field.Value == "" || len(field.Value) > 64 {
				continue
			}

			if f, ok := i.KeyFacets[field.Id]; ok {
				//if !f.BaseField.HideFacet {
				f.AddValueLink(field.Value, item.Id)
				//}
			}
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
	i.mu.Lock()
	defer i.mu.Unlock()
	i.UpsertItemUnsafe(item)
}

func (i *Index) Lock() {
	i.mu.Lock()
}

func (i *Index) Unlock() {
	i.mu.Unlock()
}

func (i *Index) UpsertItemUnsafe(item *DataItem) {

	current, isUpdate := i.Items[item.Id]
	if isUpdate {
		i.removeItemValues(current)
	} else {
		go i.AutoSuggest.InsertItem(item)
	}
	i.AllItems[item.Id] = &item.ItemFields
	i.addItemValues(item)

	i.Items[item.Id] = item
	if i.Search != nil {
		go i.Search.CreateDocument(item.Id, item.Title)
	}

	if i.ChangeHandler != nil {
		if isUpdate {
			i.ChangeHandler.ItemChanged(item)
		} else {
			i.ChangeHandler.ItemAdded(item)
		}
	}
}

func (i *Index) DeleteItem(id uint) {
	item, ok := i.Items[id]
	if ok {
		i.removeItemValues(item)
		delete(i.Items, id)
		delete(i.AllItems, id)
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
			items[idx] = ResultItem{
				BaseItem: &item.BaseItem,
				Fields:   item.getFieldValues(),
			}
			idx++
		}
	}
	return items[0:idx]
}
