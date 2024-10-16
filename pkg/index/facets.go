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
	l := uint(len(*ids))
	needsTruncation := l > 6144
	if sortIndex == nil {
		log.Println("no sort index for fields")
		return Facets{
			Fields:       []JsonKeyResult{},
			NumberFields: []JsonNumberResult{},
			IntFields:    []JsonNumberResult{},
		}
	}
	// Preallocate slices for fields, numberFields, and intFields to avoid repeated allocations
	fields := make(map[uint]map[string]uint, len(i.KeyFacets))
	numberFields := make(map[uint]*NumberResult[float64], len(i.DecimalFacets))
	intFields := make(map[uint]*NumberResult[int], len(i.IntFacets))

	// Use a single loop to initialize ignored fields and prepopulate maps
	for key, facet := range i.KeyFacets {
		if facet.HideFacet || (facet.Priority < 256 && needsTruncation) || (facet.CategoryLevel > 0) {
			//	ignoredKeyFields[key] = struct{}{}
		} else {
			fields[key] = make(map[string]uint)
		}

	}

	for key, facet := range i.IntFacets {
		if facet.HideFacet || (facet.Priority < 29176 && needsTruncation) {
		} else {
			intFields[key] = &NumberResult[int]{}
		}

	}

	for key, facet := range i.DecimalFacets {
		if facet.HideFacet || (facet.Priority < 29176 && needsTruncation) {
		} else {
			numberFields[key] = &NumberResult[float64]{}
		}

	}

	for id := range *ids {

		item, ok := i.Items[id]
		if !ok {
			continue
		}
		if item.Fields != nil {
			for _, field := range item.Fields {
				if f, ok := fields[field.Id]; ok {
					f[field.Value]++
					//f.AddValue(&field.Value) // TODO optimize
				}
			}
		}
		if item.DecimalFields != nil {
			for _, field := range item.DecimalFields {
				if f, ok := numberFields[field.Id]; ok {
					f.AddValue(field.Value)
				}
			}
		}
		if item.IntegerFields != nil {
			for _, field := range item.IntegerFields {
				if f, ok := intFields[field.Id]; ok {
					f.AddValue(field.Value)
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
