package index

import (
	"tornberg.me/facet-search/pkg/facet"
)

type KeyResult struct {
	values map[string]int
}

type JsonKeyResult struct {
	*facet.BaseField
	Values *map[string]uint `json:"values"`
}

// func (k *KeyResult) AddValue(value *string) {
// 	if count, ok := k.values[*value]; ok {
// 		k.values[*value] = count + 1
// 	} else {
// 		k.values[*value] = 1
// 	}
// }

func (k *KeyResult) GetValues() map[string]int {
	return k.values
}

type NumberResult[V float64 | int] struct {
	//*facet.BaseField
	Count uint
	Min   V
	Max   V
}

type JsonNumberResult struct {
	*facet.BaseField
	Count uint        `json:"count"`
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

func (i *Index) GetFacetsFromResult(ids *facet.MatchList, filters *Filters, sortIndex *facet.SortIndex) Facets {

	if sortIndex == nil {
		return Facets{
			Fields:       []JsonKeyResult{},
			NumberFields: []JsonNumberResult{},
			IntFields:    []JsonNumberResult{},
		}
	}

	fields := map[uint]map[string]uint{}
	numberFields := map[uint]*NumberResult[float64]{}
	intFields := map[uint]*NumberResult[int]{}

	//ignoredKeyFields := map[uint]struct{}{}

	// if filters != nil {
	// 	for _, filter := range filters.StringFilter {
	// 		ignoredKeyFields[filter.Id] = struct{}{}
	// 	}

	// }

	// if len(i.DefaultFacets.Fields) > 0 && len(*ids) > 65535 {
	// 	for id, _ := range i.KeyFacets {
	// 		ignoredKeyFields[id] = struct{}{}

	// 	}
	// 	for _, defaultField := range i.DefaultFacets.Fields[0:min(len(i.DefaultFacets.Fields), 10)] {
	// 		delete(ignoredKeyFields, defaultField.Id)
	// 	}

	// }
	// if (len(ignoredKeyFields) + len(ignoredDecimalFields) + len(ignoredIntFields)) == 0 {
	// 	return i.DefaultFacets
	// }
	for _, itemFields := range *ids {

		//item, ok := i.Items[id]
		// if !ok {
		// 	continue
		// }
		if itemFields.Fields != nil {
			for _, field := range itemFields.Fields {

				l := len(field.Value)
				if l == 0 || l > 64 {
					continue
				}
				// if _, ok := ignoredKeyFields[field.Id]; ok {
				// 	continue
				// }

				if f, ok := fields[field.Id]; ok {
					f[field.Value]++
					//f.AddValue(&field.Value) // TODO optimize
				} else {
					//count++

					fields[field.Id] = map[string]uint{
						field.Value: 1,
					}

				}
			}
		}
		if itemFields.DecimalFields != nil {
			for _, field := range itemFields.DecimalFields {

				if f, ok := numberFields[field.Id]; ok {
					f.AddValue(field.Value)
				} else {
					//count++
					numberFields[field.Id] = &NumberResult[float64]{
						Count: 1,
						Min:   field.Value,
						Max:   field.Value,
					}
				}
			}
		}
		if itemFields.IntegerFields != nil {
			for _, field := range itemFields.IntegerFields {
				if field.Value == 0 || field.Value == -1 {
					continue
				}

				if f, ok := intFields[field.Id]; ok {
					f.AddValue(field.Value)
				} else {
					//count++
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
