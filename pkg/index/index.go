package index

import (
	"log"
	"sort"
	"time"

	"tornberg.me/facet-search/pkg/facet"
)

type Index struct {
	KeyFacets     map[int64]*facet.Field[string]
	DecimalFacets map[int64]*facet.NumberField[float64]
	IntFacets     map[int64]*facet.NumberField[int]
	BoolFacets    map[int64]*facet.Field[bool]
	Items         map[int64]Item
}

func NewIndex() *Index {
	return &Index{
		KeyFacets:     make(map[int64]*facet.Field[string]),
		DecimalFacets: make(map[int64]*facet.NumberField[float64]),
		IntFacets:     make(map[int64]*facet.NumberField[int]),
		BoolFacets:    make(map[int64]*facet.Field[bool]),
		Items:         make(map[int64]Item),
	}
}

func (i *Index) AddKeyField(field *facet.Field[string]) {
	i.KeyFacets[field.Id] = field
}

func (i *Index) AddBoolField(field *facet.Field[bool]) {
	i.BoolFacets[field.Id] = field
}

func (i *Index) AddDecimalField(field *facet.NumberField[float64]) {
	i.DecimalFacets[field.Id] = field
}

func (i *Index) AddIntegerField(field *facet.NumberField[int]) {
	i.IntFacets[field.Id] = field
}

func (i *Index) AddItemValues(item Item) {

	for key, value := range item.Fields {
		if f, ok := i.KeyFacets[key]; ok {
			f.AddValueLink(value, item.Id)
		} else {
			//delete(item.Fields, key)
			log.Printf("Field not found %v: %v", key, value)
		}
	}
	for key, value := range item.DecimalFields {
		if f, ok := i.DecimalFacets[key]; ok {
			f.AddValueLink(value, item.Id)
		} else {
			log.Printf("DecimalField not found %v: %v", key, value)
			//delete(item.NumberFields, key)
		}
	}

	for key, value := range item.IntegerFields {
		if f, ok := i.IntFacets[key]; ok {
			f.AddValueLink(value, item.Id)
		} else {
			log.Printf("IntField not found %v: %v", key, value)
			//delete(item.NumberFields, key)
		}
	}

	for key, value := range item.BoolFields {
		if f, ok := i.BoolFacets[key]; ok {
			f.AddValueLink(value, item.Id)
		} else {
			log.Printf("BoolField not found %v: %v", key, value)
			//delete(item.BoolFields, key)
		}
	}
}

func (i *Index) AddItem(item Item) {
	i.Items[item.Id] = item
	i.AddItemValues(item)
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

type StringResult[V string | bool] struct {
	Field  *facet.Field[V] `json:"field"`
	Values map[string]int  `json:"values"`
}

type NumberResult struct {
	Field *facet.NumberField[float64] `json:"field"`
	Count int                         `json:"count"`
	Min   float64                     `json:"min"`
	Max   float64                     `json:"max"`
}

type Facets struct {
	Fields       []StringResult[string] `json:"fields"`
	NumberFields []*NumberResult        `json:"numberFields"`
	BoolFields   []StringResult[bool]   `json:"boolFields"`
}

func (i *Index) GetFacetsFromResult(result *facet.Result, filters *Filters, sortIndex *facet.SortIndex) Facets {
	start := time.Now()
	all := result.Ids()
	ids := all[0:min(1000, len(all))]

	fields := map[int64]*StringResult[string]{}
	numberFields := map[int64]*NumberResult{}
	boolFields := map[int64]*StringResult[bool]{}

	for _, id := range ids {
		item, ok := i.Items[id] // todo optimize and maybe sort in this step
		if !ok {
			continue
		}
		for key, value := range item.Fields {
			if f, ok := fields[key]; ok {
				f.Values[value]++
			} else {
				if baseField, ok := i.KeyFacets[key]; ok {
					fields[key] = &StringResult[string]{
						Field:  baseField,
						Values: map[string]int{value: 1},
					}
				}
			}
		}
		for key, value := range item.DecimalFields {
			if f, ok := numberFields[key]; ok {
				if value < f.Min {
					f.Min = value
				} else if value > f.Max {
					f.Max = value
				}
				f.Count++
			} else {
				if baseField, ok := i.DecimalFacets[key]; ok {
					numberFields[key] = &NumberResult{
						Field: baseField,
						Count: 1,
						Min:   value,
						Max:   value,
					}
				}
			}
		}
		for key, value := range item.BoolFields {
			if f, ok := boolFields[key]; ok {
				f.Values[stringValue(value)]++
			} else {
				if baseField, ok := i.BoolFacets[key]; ok {
					boolFields[key] = &StringResult[bool]{
						Field:  baseField,
						Values: map[string]int{stringValue(value): 1},
					}
				}
			}
		}
	}
	log.Printf("GetFacetsFromResultIds took %v", time.Since(start))

	return Facets{
		Fields:       mapToSlice[string](fields, sortIndex),
		NumberFields: mapToSliceRef(numberFields, sortIndex),
		BoolFields:   mapToSlice[bool](boolFields, sortIndex),
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

func (i *Index) Match(search *Filters) facet.Result {

	results := make(chan facet.Result)
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
