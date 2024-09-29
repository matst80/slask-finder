package index

import (
	"log"

	"tornberg.me/facet-search/pkg/facet"
)

type KeyResult struct {
	values map[string]int
}

type JsonKeyResult struct {
	*facet.BaseField
	Values *map[string]uint `json:"values"`
}

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

func (i *Index) GetFacetsFromResult(ids *facet.IdList, filters *Filters, sortIndex *facet.SortIndex) Facets {
	needsTruncation := len(*ids) > 6144
	if sortIndex == nil {
		log.Println("no sort index for fields")
		return Facets{
			Fields:       []JsonKeyResult{},
			NumberFields: []JsonNumberResult{},
			IntFields:    []JsonNumberResult{},
		}
	}

	fields := make(map[uint]map[string]uint)
	numberFields := make(map[uint]*NumberResult[float64])
	intFields := make(map[uint]*NumberResult[int])

	ignoredKeyFields := make(map[uint]struct{})
	ignoredIntFields := make(map[uint]struct{})
	ignoredDecimalFields := make(map[uint]struct{})

	for key, facet := range i.KeyFacets {
		if facet.HideFacet || (facet.Priority < 256 && needsTruncation) {
			ignoredKeyFields[key] = struct{}{}
		}
	}

	for key, facet := range i.IntFacets {
		if facet.HideFacet || (facet.Priority < 29176 && needsTruncation) {
			ignoredIntFields[key] = struct{}{}
		}
	}

	for key, facet := range i.DecimalFacets {
		if facet.HideFacet || (facet.Priority < 29176 && needsTruncation) {
			ignoredDecimalFields[key] = struct{}{}
		}
	}

	for id := range *ids {

		item, ok := i.AllItems[id]
		if !ok {
			continue
		}
		if item.Fields != nil {
			for _, field := range item.Fields {

				l := len(field.Value)
				if l == 0 || l > 64 {
					continue
				}
				if _, ok := ignoredKeyFields[field.Id]; ok {
					continue
				}

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
		if item.DecimalFields != nil {
			for _, field := range item.DecimalFields {
				if _, ok := ignoredKeyFields[field.Id]; ok {
					continue
				}
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
		if item.IntegerFields != nil {
			for _, field := range item.IntegerFields {

				if field.Value == 0 || field.Value == -1 {
					continue
				}

				if _, ok := ignoredKeyFields[field.Id]; ok {
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
