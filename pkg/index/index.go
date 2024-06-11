package index

import (
	"log"
	"sort"
	"time"

	"tornberg.me/facet-search/pkg/facet"
)

type Index struct {
	KeyFacets     map[int64]*facet.KeyField[string]
	DecimalFacets map[int64]*facet.NumberField[float64]
	IntFacets     map[int64]*facet.NumberField[int]
	BoolFacets    map[int64]*facet.KeyField[bool]
	Items         map[int64]Item
}

func NewIndex() *Index {
	return &Index{
		KeyFacets:     make(map[int64]*facet.KeyField[string]),
		DecimalFacets: make(map[int64]*facet.NumberField[float64]),
		IntFacets:     make(map[int64]*facet.NumberField[int]),
		BoolFacets:    make(map[int64]*facet.KeyField[bool]),
		Items:         make(map[int64]Item),
	}
}

func (i *Index) AddKeyField(field *facet.BaseField) {
	i.KeyFacets[field.Id] = facet.EmptyKeyValueField[string](field)
}

func (i *Index) AddBoolField(field *facet.BaseField) {
	i.BoolFacets[field.Id] = facet.EmptyKeyValueField[bool](field)
}

func (i *Index) AddDecimalField(field *facet.BaseField) {
	i.DecimalFacets[field.Id] = facet.EmptyNumberField[float64](field)
}

func (i *Index) AddIntegerField(field *facet.BaseField) {
	i.IntFacets[field.Id] = facet.EmptyNumberField[int](field)
}

func (i *Index) AddItemValues(item DataItem) {

	for key, value := range item.Fields {
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
		if f, ok := i.IntFacets[key]; ok {
			f.AddValueLink(value, item.Id)
		} else {
			//log.Printf("IntField not found %v: %v", key, value)
			//delete(item.NumberFields, key)
		}
	}

	for key, value := range item.BoolFields {
		if f, ok := i.BoolFacets[key]; ok {
			f.AddValueLink(value, item.Id)
		} else {
			//log.Printf("BoolField not found %v: %v", key, value)
			//delete(item.BoolFields, key)
		}
	}
}

func getFields[K facet.FieldKeyValue](fields map[int64]*facet.KeyField[K], itemFields map[int64]K) map[int64]ItemKeyField[K] {
	newFields := make(map[int64]ItemKeyField[K])
	for key, value := range itemFields {
		if f, ok := fields[key]; ok {
			newFields[key] = ItemKeyField[K]{KeyField: f, Value: value}
		}
	}
	return newFields
}

func getNumberFields[K facet.FieldNumberValue](fields map[int64]*facet.NumberField[K], itemFields map[int64]K) map[int64]ItemNumberField[K] {
	newFields := make(map[int64]ItemNumberField[K])
	for key, value := range itemFields {
		if f, ok := fields[key]; ok {
			newFields[key] = ItemNumberField[K]{NumberField: f, Value: value}
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
		BoolFields:    getFields(i.BoolFacets, item.BoolFields),
	}
}

func (i *Index) GetItem(id int64) Item {
	return i.Items[id]
}

func (i *Index) HasItem(id int64) bool {
	return i.Items[id].Id == id
}

func (i *Index) GetItemIds(ids []int64, page int, pageSize int) []int64 {
	l := len(ids)
	start := page * pageSize
	end := min(l, start+pageSize)
	if start > l {
		return ids[0:0]
	}
	return ids[start:end]
}

func (i *Index) GetItems(ids []int64, page int, pageSize int) []*Item {
	items := make([]*Item, min(len(ids), pageSize))
	idx := 0
	for _, id := range i.GetItemIds(ids, page, pageSize) {
		item, ok := i.Items[id]
		if ok {
			items[idx] = &item
			idx++
		}
	}
	return items[0:idx]
}

type KeyResult[K string | bool] struct {
	*facet.BaseField
	Values map[K]int `json:"values"`
}

type NumberResult[V float64 | int] struct {
	*facet.BaseField
	Count int `json:"count"`
	Min   V   `json:"min"`
	Max   V   `json:"max"`
}

type Facets struct {
	Fields       []KeyResult[string]     `json:"fields"`
	NumberFields []NumberResult[float64] `json:"numberFields"`
	BoolFields   []KeyResult[string]     `json:"boolFields"`
}

func (i *Index) GetFacetsFromResult(ids *facet.IdList, filters *Filters, sortIndex *facet.SortIndex) Facets {
	start := time.Now()
	count := 0
	fields := map[int64]*KeyResult[string]{}
	numberFields := map[int64]*NumberResult[float64]{}
	boolFields := map[int64]*KeyResult[bool]{}

	for id := range *ids {
		item, ok := i.Items[id] // todo optimize and maybe sort in this step
		if !ok {
			continue
		}
		count++
		if count > 1024 {
			break
		}
		for fieldId, field := range item.Fields {
			if f, ok := fields[fieldId]; ok {
				f.Values[field.Value]++
			} else {
				fields[fieldId] = &KeyResult[string]{
					BaseField: field.BaseField,
					Values:    map[string]int{field.Value: 1},
				}
			}
		}

		for key, field := range item.DecimalFields {
			if f, ok := numberFields[key]; ok {
				if field.Value < f.Min {
					f.Min = field.Value
				} else if field.Value > f.Max {
					f.Max = field.Value
				}
				f.Count++
			} else {
				numberFields[key] = &NumberResult[float64]{
					BaseField: field.BaseField,
					Count:     1,
					Min:       field.Value,
					Max:       field.Value,
				}
			}
		}
		for fieldId, field := range item.Fields {
			if f, ok := fields[fieldId]; ok {
				f.Values[field.Value]++
			} else {
				fields[fieldId] = &KeyResult[string]{
					BaseField: field.BaseField,
					Values:    map[string]int{field.Value: 1},
				}
			}
		}
	}
	log.Printf("GetFacetsFromResultIds took %v", time.Since(start))
	b := boolToStringResult(boolFields)
	return Facets{
		Fields:       mapToSlice(fields, sortIndex),
		NumberFields: mapToSliceNumber(numberFields, sortIndex),
		BoolFields:   mapToSlice(b, sortIndex),
	}
}

type NumberSearch struct {
	Id  int64   `json:"id"`
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

type StringSearch struct {
	Id    int64  `json:"id"`
	Value string `json:"value"`
}

type BoolSearch struct {
	Id    int64 `json:"id"`
	Value bool  `json:"value"`
}

type Filters struct {
	StringFilter []StringSearch `json:"string"`
	NumberFilter []NumberSearch `json:"number"`
	BoolFilter   []BoolSearch   `json:"bool"`
}

func (i *Index) MakeSortForFields() facet.SortIndex {

	l := len(i.BoolFacets) + len(i.DecimalFacets) + len(i.KeyFacets)
	idx := 0
	sortIndex := make(facet.SortIndex, l)
	sortMap := make(facet.ByValue, l)
	for _, item := range i.BoolFacets {
		sortMap[idx] = facet.Lookup{Id: item.Id, Value: float64(item.TotalCount())}
		idx++
	}
	for _, item := range i.DecimalFacets {
		sortMap[idx] = facet.Lookup{Id: item.Id, Value: float64(item.TotalCount())}
		idx++
	}
	for _, item := range i.KeyFacets {
		sortMap[idx] = facet.Lookup{Id: item.Id, Value: float64(item.TotalCount())}
		idx++
	}
	sort.Sort(sort.Reverse(sortMap))
	for idx, item := range sortMap {
		sortIndex[idx] = item.Id
	}
	return sortIndex
}

type IndexMatch struct {
	Ids    facet.IdList
	Fields facet.IdList
}

func (i *Index) Match(search *Filters) facet.IdList {

	results := make(chan facet.IdList)

	len := 0
	for _, fld := range search.BoolFilter {
		if f, ok := i.BoolFacets[fld.Id]; ok {
			len++
			go func(field BoolSearch) {

				start := time.Now()
				results <- f.Matches(field.Value)

				log.Printf("Bool match took %v", time.Since(start))

			}(fld)

		}
	}
	for _, fld := range search.StringFilter {
		if f, ok := i.KeyFacets[fld.Id]; ok {
			len++
			go func(field StringSearch) {
				start := time.Now()
				results <- f.Matches(field.Value)

				log.Printf("String match took %v", time.Since(start))

			}(fld)
		}
	}
	for _, fld := range search.NumberFilter {
		if f, ok := i.DecimalFacets[fld.Id]; ok {
			len++
			go func(field NumberSearch) {

				start := time.Now()
				results <- f.MatchesRange(field.Min, field.Max)

				log.Printf("Decimal match took %v", time.Since(start))

			}(fld)
		}
	}

	return facet.MakeIntersectResult(results, len)

}
