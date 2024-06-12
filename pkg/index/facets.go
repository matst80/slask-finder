package index

import (
	"log"
	"time"

	"tornberg.me/facet-search/pkg/facet"
)

type hashKeyResult struct {
	value string
	count int
}
type KeyResult struct {
	*facet.BaseField
	values map[uint]*hashKeyResult
}

func (k *KeyResult) GetValues() map[string]int {
	values := make(map[string]int)
	for _, v := range k.values {
		values[v.value] = v.count
	}
	return values
}

type JsonKeyResult struct {
	*facet.BaseField
	Values map[string]int `json:"values"`
}

func (k *KeyResult) AddValue(hash uint, value string) {
	if v, ok := k.values[hash]; ok {
		v.count++
	} else {
		k.values[hash] = &hashKeyResult{value: value, count: 1}
	}
	//k.Values[value]++
}

type NumberResult[V float64 | int] struct {
	*facet.BaseField
	Count int `json:"count"`
	Min   V   `json:"min"`
	Max   V   `json:"max"`
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
	Fields       []JsonKeyResult         `json:"fields"`
	NumberFields []NumberResult[float64] `json:"numberFields"`
	IntFields    []NumberResult[int]     `json:"integerFields"`
}

func (i *Index) GetFacetsFromResult(ids *facet.IdList, filters *Filters, sortIndex *facet.SortIndex) Facets {
	start := time.Now()
	count := 0
	fields := map[uint]KeyResult{}
	numberFields := map[uint]NumberResult[float64]{}
	intFields := map[uint]NumberResult[int]{}

	for id := range *ids {

		item, ok := i.Items[id]
		if !ok {
			continue
		}

		for fieldId, field := range item.Fields {
			if field.field.BaseField.HideFacet || field.Value == "" {
				continue
			}
			if f, ok := fields[fieldId]; ok {
				f.AddValue(field.ValueHash, field.Value) // TODO optimize
			} else {
				count++

				fields[fieldId] = KeyResult{
					BaseField: field.field.BaseField,
					values:    map[uint]*hashKeyResult{field.ValueHash: {value: field.Value, count: 1}},
				}
			}
		}

		for key, field := range item.DecimalFields {
			if f, ok := numberFields[key]; ok {
				f.AddValue(field.Value)
			} else {
				count++
				numberFields[key] = NumberResult[float64]{
					BaseField: field.field.BaseField,
					Count:     1,
					Min:       field.Value,
					Max:       field.Value,
				}
			}
		}
		for key, field := range item.IntegerFields {
			if f, ok := intFields[key]; ok {
				f.AddValue(field.Value)
			} else {
				count++
				intFields[key] = NumberResult[int]{
					BaseField: field.field.BaseField,
					Count:     1,
					Min:       field.Value,
					Max:       field.Value,
				}
			}
		}

	}
	log.Printf("GetFacetsFromResultIds took %v, found %d facets", time.Since(start), len(fields)+len(numberFields)+len(intFields))

	return Facets{
		Fields:       mapToSlice(fields, sortIndex),
		NumberFields: mapToSliceNumber(numberFields, sortIndex),
		IntFields:    mapToSliceNumber(intFields, sortIndex),
	}
}
