package index

import (
	"sort"

	"tornberg.me/facet-search/pkg/facet"
)

type Index struct {
	KeyFacets     map[uint]*facet.KeyField
	DecimalFacets map[uint]*facet.NumberField[float64]
	IntFacets     map[uint]*facet.NumberField[int]

	Items map[uint]Item
}

func NewIndex() *Index {
	return &Index{
		KeyFacets:     make(map[uint]*facet.KeyField),
		DecimalFacets: make(map[uint]*facet.NumberField[float64]),
		IntFacets:     make(map[uint]*facet.NumberField[int]),

		Items: make(map[uint]Item),
	}
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

func (i *Index) AddItemValues(item DataItem) {

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

func getFields(fields map[uint]*facet.KeyField, itemFields map[uint]string) map[uint]ItemKeyField {
	newFields := make(map[uint]ItemKeyField)
	for key, value := range itemFields {
		if value == "" {
			continue
		}
		if f, ok := fields[key]; ok {
			newFields[key] = ItemKeyField{field: f, Value: value, ValueHash: HashString(value)}
		}
	}
	return newFields
}

func getNumberFields[K facet.FieldNumberValue](fields map[uint]*facet.NumberField[K], itemFields map[uint]K) map[uint]ItemNumberField[K] {
	newFields := make(map[uint]ItemNumberField[K])
	for key, value := range itemFields {
		if f, ok := fields[key]; ok {
			newFields[key] = ItemNumberField[K]{field: f, Value: value}
		}
	}
	return newFields
}

func (i *Index) AddItem(item DataItem) {

	i.AddItemValues(item)
	i.Items[item.Id] = Item{
		BaseItem:      item.BaseItem,
		Fields:        getFields(i.KeyFacets, item.Fields),
		DecimalFields: getNumberFields(i.DecimalFacets, item.DecimalFields),
		IntegerFields: getNumberFields(i.IntFacets, item.IntegerFields),
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

func (i *Index) MakeSortForFields() facet.SortIndex {

	l := len(i.DecimalFacets) + len(i.KeyFacets) + len(i.IntFacets)
	idx := 0
	sortIndex := make(facet.SortIndex, l)
	sortMap := make(facet.ByValue, l)

	for _, item := range i.DecimalFacets {
		sortMap[idx] = facet.Lookup{Id: item.Id, Value: float64(item.TotalCount())}
		idx++
	}
	for _, item := range i.KeyFacets {
		sortMap[idx] = facet.Lookup{Id: item.Id, Value: float64(item.TotalCount())}
		idx++
	}
	for _, item := range i.IntFacets {
		sortMap[idx] = facet.Lookup{Id: item.Id, Value: float64(item.TotalCount())}
		idx++
	}
	sort.Sort(sort.Reverse(sortMap))
	for idx, item := range sortMap {
		sortIndex[idx] = item.Id
	}
	return sortIndex
}
