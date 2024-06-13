package index

import (
	"tornberg.me/facet-search/pkg/facet"
)

type KeyFacet = facet.KeyField
type DecimalFacet = facet.NumberField[float64]
type IntFacet = facet.NumberField[int]

type ChangeHandler interface {
	ItemChanged(item DataItem)
	ItemDeleted(id uint)
	ItemAdded(item DataItem)
}

type Index struct {
	KeyFacets     map[uint]*KeyFacet
	DecimalFacets map[uint]*DecimalFacet
	IntFacets     map[uint]*IntFacet
	DefaultFacets Facets
	Items         map[uint]Item
	AllItems      facet.IdList
	ChangeHandler ChangeHandler
}

func NewIndex() *Index {
	return &Index{
		KeyFacets:     make(map[uint]*KeyFacet),
		DecimalFacets: make(map[uint]*DecimalFacet),
		IntFacets:     make(map[uint]*IntFacet),
		Items:         make(map[uint]Item),
		AllItems:      facet.IdList{},
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
		if f, ok := i.KeyFacets[key]; ok {
			f.AddValueLink(value, item.Id)
		} else {
			//delete(item.Fields, key)
			//log.Printf("Field not found %v: %v", key, value)
		}
	}
	for key, value := range item.DecimalFields {
		if f, ok := i.DecimalFacets[key]; ok {
			f.AddValueLink(value, item.Id)
		} else {
			//log.Printf("DecimalField not found %v: %v", key, value)
			//delete(item.NumberFields, key)
		}
	}

	for key, value := range item.IntegerFields {
		if value == 0 {
			continue
		}
		if f, ok := i.IntFacets[key]; ok {
			f.AddValueLink(value, item.Id)
		} else {
			//log.Printf("IntField not found %v: %v", key, value)
			//delete(item.NumberFields, key)
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

func (i *Index) UpsertItem(item DataItem) {
	current, isUpdate := i.Items[item.Id]
	if isUpdate {
		i.removeItemValues(current)
	}
	i.AllItems[item.Id] = struct{}{}
	i.addItemValues(item)
	i.Items[item.Id] = Item{
		BaseItem:      item.BaseItem,
		Fields:        getFields(item.Fields),
		DecimalFields: getNumberFields(item.DecimalFields),
		IntegerFields: getNumberFields(item.IntegerFields),
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

func (i *Index) GetItem(id uint) Item {
	return i.Items[id]
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

func getFieldValues(item *Item) map[uint]interface{} {
	fields := map[uint]interface{}{}
	for key, value := range item.Fields {
		fields[key] = value.Value
	}
	for key, value := range item.DecimalFields {
		fields[key] = value.Value
	}
	for key, value := range item.IntegerFields {
		fields[key] = value.Value
	}

	return fields
}

func (i *Index) GetItems(ids []uint, page int, pageSize int) []ResultItem {
	items := make([]ResultItem, min(len(ids), pageSize))
	idx := 0
	for _, id := range i.GetItemIds(ids, page, pageSize) {
		item, ok := i.Items[id]
		if ok {
			items[idx] = ResultItem{
				BaseItem: item.BaseItem,
				Fields:   getFieldValues(&item),
			}
			idx++
		}
	}
	return items[0:idx]
}
