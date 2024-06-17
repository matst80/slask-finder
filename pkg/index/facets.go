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
	for id := range *ids {

		item, ok := i.Items[id]
		if !ok {
			continue
		}

		for fieldId, value := range item.Fields {
			if _, ok := ignoredKeyFields[fieldId]; ok {
				continue
			}
			if f, ok := fields[fieldId]; ok {
				f.AddValue(&value) // TODO optimize
			} else {
				count++

				fields[fieldId] = &KeyResult{
					values: map[string]int{
						value: 1,
					},
				}
			}
		}

		for key, field := range item.DecimalFields {
			if _, ok := ignoredDecimalFields[key]; ok {
				continue
			}
			if f, ok := numberFields[key]; ok {
				f.AddValue(field)
			} else {
				count++
				numberFields[key] = &NumberResult[float64]{
					Count: 1,
					Min:   field,
					Max:   field,
				}
			}
		}
		for key, field := range item.IntegerFields {
			if _, ok := ignoredIntFields[key]; ok {
				continue
			}
			if f, ok := intFields[key]; ok {
				f.AddValue(field)
			} else {
				count++
				intFields[key] = &NumberResult[int]{
					Count: 1,
					Min:   field,
					Max:   field,
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
