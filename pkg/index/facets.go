package index

import (
	"tornberg.me/facet-search/pkg/facet"
)

type KeyResult struct {
	values map[string]int
}

type JsonKeyResult struct {
	*facet.BaseField
	Values map[string]int `json:"values"`
}

func (k *KeyResult) AddValue(value *string) {
	if count, ok := k.values[*value]; ok {
		k.values[*value] = count + 1
	} else {
		k.values[*value] = 1
	}
}

func (k *KeyResult) GetValues() map[string]int {
	return k.values
}

type NumberResult[V float64 | int] struct {
	//*facet.BaseField
	Count int
	Min   V
	Max   V
}

type JsonNumberResult struct {
	*facet.BaseField
	Count int         `json:"count"`
	Min   interface{} `json:"min"`
	Max   interface{} `json:"max"`
}

func (k *NumberResult[V]) AddValue(value V) {
	if value < k.Min {
		k.Min = value
	} else if value > k.Max {
		k.Max = value
	}
	k.Count++
}

type Facets struct {
	Fields       []JsonKeyResult    `json:"fields"`
	NumberFields []JsonNumberResult `json:"numberFields"`
	IntFields    []JsonNumberResult `json:"integerFields"`
}

func (i *Index) GetFacetsFromResult(ids *facet.IdList, filters *Filters, sortIndex *facet.SortIndex) Facets {

	if sortIndex == nil {
		return Facets{
			Fields:       []JsonKeyResult{},
			NumberFields: []JsonNumberResult{},
			IntFields:    []JsonNumberResult{},
		}
	}
	count := 0
	fields := map[uint]*KeyResult{}
	numberFields := map[uint]*NumberResult[float64]{}
	intFields := map[uint]*NumberResult[int]{}

	ignoredKeyFields := map[uint]struct{}{}
	ignoredDecimalFields := map[uint]struct{}{}
	ignoredIntFields := map[uint]struct{}{}
	if filters != nil {
		for _, filter := range filters.StringFilter {
			ignoredKeyFields[filter.Id] = struct{}{}
		}

		for _, filter := range filters.NumberFilter {
			ignoredDecimalFields[filter.Id] = struct{}{}
		}

		for _, filter := range filters.IntegerFilter {
			ignoredIntFields[filter.Id] = struct{}{}
		}
	}
	// if (len(ignoredKeyFields) + len(ignoredDecimalFields) + len(ignoredIntFields)) == 0 {
	// 	return i.DefaultFacets
	// }
	for id := range *ids {

		item, ok := i.Items[id]
		if !ok {
			continue
		}
		if item.Fields != nil {
			for _, field := range *item.Fields {
				if _, ok := ignoredKeyFields[field.Id]; ok {
					continue
				}
				if f, ok := fields[field.Id]; ok {
					l := len(field.Value)
					if l == 0 || l > 64 {
						continue
					}
					f.AddValue(&field.Value) // TODO optimize
				} else {
					count++

					fields[field.Id] = &KeyResult{
						values: map[string]int{
							field.Value: 1,
						},
					}
				}
			}
		}
		if item.DecimalFields != nil {
			for _, field := range *item.DecimalFields {
				if _, ok := ignoredDecimalFields[field.Id]; ok {
					continue
				}
				if f, ok := numberFields[field.Id]; ok {
					f.AddValue(field.Value)
				} else {
					count++
					numberFields[field.Id] = &NumberResult[float64]{
						Count: 1,
						Min:   field.Value,
						Max:   field.Value,
					}
				}
			}
		}
		if item.IntegerFields != nil {
			for _, field := range *item.IntegerFields {
				if _, ok := ignoredIntFields[field.Id]; ok {
					continue
				}
				if f, ok := intFields[field.Id]; ok {
					f.AddValue(field.Value)
				} else {
					count++
					intFields[field.Id] = &NumberResult[int]{
						Count: 1,
						Min:   field.Value,
						Max:   field.Value,
					}
				}
			}
		}
	}

	return Facets{
		Fields:       i.mapToSlice(fields, sortIndex),
		NumberFields: mapToSliceNumber(i.DecimalFacets, numberFields, sortIndex),
		IntFields:    mapToSliceNumber(i.IntFacets, intFields, sortIndex),
	}
}
