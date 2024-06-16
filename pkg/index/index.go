package index

import (
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
	KeyFacets     map[uint]*KeyFacet
	DecimalFacets map[uint]*DecimalFacet
	IntFacets     map[uint]*IntFacet
	DefaultFacets Facets
	Items         map[uint]Item
	AllItems      facet.IdList
	AutoSuggest   AutoSuggest
	ChangeHandler ChangeHandler
	Search        *search.FreeTextIndex
}

func NewIndex(freeText *search.FreeTextIndex) *Index {
	return &Index{
		KeyFacets:     make(map[uint]*KeyFacet),
		DecimalFacets: make(map[uint]*DecimalFacet),
		IntFacets:     make(map[uint]*IntFacet),
		Items:         make(map[uint]Item),
		AllItems:      facet.IdList{},
		AutoSuggest:   AutoSuggest{Trie: search.NewTrie()},
		Search:        freeText,
	}
}

func (i *Index) CreateDefaultFacets(sort *facet.SortIndex) {
	i.DefaultFacets = i.GetFacetsFromResult(&i.AllItems, &Filters{}, sort)
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

func (i *Index) addItemValues(item DataItem) {

	for key, value := range item.Fields {
		if value == "" {
			continue
		}
		v := value
		if f, ok := i.KeyFacets[key]; ok {
			f.AddValueLink(v, item.Id)
		}
	}
	for key, value := range item.DecimalFields {
		if value == 0.0 {
			continue
		}
		if f, ok := i.DecimalFacets[key]; ok {
			f.AddValueLink(value, item.Id)
		}
	}

	for key, value := range item.IntegerFields {
		if value == 0 {
			continue
		}
		if f, ok := i.IntFacets[key]; ok {
			f.AddValueLink(value, item.Id)
		}
	}

}

func (i *Index) removeItemValues(item Item) {
	for key, value := range item.Fields {
		if f, ok := i.KeyFacets[key]; ok {
			f.RemoveValueLink(*value.Value, item.Id)
		}
	}
	for key, value := range item.DecimalFields {
		if f, ok := i.DecimalFacets[key]; ok {
			f.RemoveValueLink(value.Value, item.Id)
		}
	}
	for key, value := range item.IntegerFields {
		if f, ok := i.IntFacets[key]; ok {
			f.RemoveValueLink(value.Value, item.Id)
		}
	}
}

func getFields(itemFields map[uint]string) map[uint]ItemKeyField {
	newFields := make(map[uint]ItemKeyField)
	for key, value := range itemFields {
		if value == "" {
			continue
		}
		newFields[key] = ItemKeyField{Value: &value}

	}
	return newFields
}

func getNumberFields[K facet.FieldNumberValue](itemFields map[uint]K) map[uint]ItemNumberField[K] {
	newFields := make(map[uint]ItemNumberField[K])
	for key, value := range itemFields {
		if value == K(-1) {
			continue
		}
		newFields[key] = ItemNumberField[K]{Value: value}
	}
	return newFields
}

func (i *Index) UpsertItem(item *DataItem) {
	current, isUpdate := i.Items[item.Id]
	if isUpdate {
		i.removeItemValues(current)
	} else {
		i.AutoSuggest.InsertItem(item)
	}
	i.AllItems[item.Id] = struct{}{}
	i.addItemValues(*item)
	keyFields := getFields(item.Fields)
	decimalFields := getNumberFields(item.DecimalFields)
	intFields := getNumberFields(item.IntegerFields)
	i.Items[item.Id] = Item{
		BaseItem: &BaseItem{
			Id:    item.Id,
			Title: item.Title,
			Sku:   item.Sku,
			Props: item.Props,
		},
		Fields:        keyFields,
		DecimalFields: decimalFields,
		IntegerFields: intFields,
		//fieldValues:   getFieldValues(item),
	}
	if i.Search != nil {
		i.Search.CreateDocument(item.Id, item.Title)
	}
	// for fieldId, field := range keyFields {
	// 	f, ok := i.itemKeyFacets[fieldId]
	// 	if !ok {
	// 		i.itemKeyFacets[fieldId] = map[uint]*ItemKeyField{
	// 			item.Id: &field,
	// 		}
	// 	} else {
	// 		f[item.Id] = &field
	// 	}
	// }
	// for fieldId, field := range decimalFields {
	// 	f, ok := i.itemNumberFacets[fieldId]
	// 	if !ok {
	// 		i.itemNumberFacets[fieldId] = map[uint]*ItemNumberField[float64]{
	// 			item.Id: &field,
	// 		}
	// 	} else {
	// 		f[item.Id] = &field
	// 	}
	// }
	// for fieldId, field := range intFields {
	// 	f, ok := i.itemIntFacets[fieldId]
	// 	if !ok {
	// 		i.itemIntFacets[fieldId] = map[uint]*ItemNumberField[int]{
	// 			item.Id: &field,
	// 		}
	// 	} else {
	// 		f[item.Id] = &field
	// 	}
	// }

	if i.ChangeHandler != nil {
		if isUpdate {
			i.ChangeHandler.ItemChanged(item)
		} else {
			i.ChangeHandler.ItemAdded(item)
		}
	}
	item = nil
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
				BaseItem: item.BaseItem,
				Fields:   item.getFieldValues(),
			}
			idx++
		}
	}
	return items[0:idx]
}
