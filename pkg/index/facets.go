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

const BufferSize = 256

type ValueContainer struct {
	values map[string]int `json:"values"`
}

func (v *ValueContainer) GetValues() map[string]int {
	return v.values
}

type KeyResult struct {
	//*facet.BaseField
	//id     uint
	buffer []string
	idx    int
	*ValueContainer
}

func (v *ValueContainer) AddValues(values []string) {
	for _, value := range values {
		if val, ok := v.values[value]; !ok {
			v.values[value] = 1
		} else {
			val++
		}
	}
}

type JsonKeyResult struct {
	*facet.BaseField
	Values map[string]int `json:"values"`
}

func (k *KeyResult) AddValue(value string) {
	k.buffer[k.idx] = value
	k.idx++

	if k.idx >= BufferSize {
		k.AddValues(k.buffer)

		k.idx = 0
	}

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
	//fieldTime := map[uint]time.Duration{}
	//s := time.Now()
	for id := range *ids {

		item, ok := i.Items[id]
		if !ok {
			continue
		}

		for fieldId, field := range item.Fields {
			//s = time.Now()

			if f, ok := fields[fieldId]; ok {
				f.AddValue(field.Value) // TODO optimize
			} else {
				count++

				fields[fieldId] = KeyResult{
					//BaseField: field.field.BaseField,
					ValueContainer: &ValueContainer{
						values: map[string]int{
							field.Value: 1,
						},
					},
					buffer: make([]string, BufferSize),
				}
			}
			//fieldTime[fieldId] += time.Since(s)
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
	go func() {
		//log.Println("Field time %v", fieldTime)
		log.Printf("GetFacetsFromResultIds took %v %v * %v ", time.Since(start), count, len(*ids))
	}()
	return Facets{
		Fields:       i.mapToSlice(fields, sortIndex),
		NumberFields: mapToSliceNumber(numberFields, sortIndex),
		IntFields:    mapToSliceNumber(intFields, sortIndex),
	}
}
