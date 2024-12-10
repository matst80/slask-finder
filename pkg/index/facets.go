package index

import (
	"github.com/matst80/slask-finder/pkg/types"
)

type Facets map[uint]FieldResult

type FieldResult interface {
	AddValue(value interface{})
	HasValues() bool
}

type KeyFieldResult struct {
	Values map[string]uint `json:"values,omitempty"`
}

func (k *KeyFieldResult) AddValue(input interface{}) {
	value, ok := input.(string)
	if !ok || value == "" {
		return
	}
	k.Values[value]++
}

func (k *KeyFieldResult) HasValues() bool {
	return len(k.Values) > 0
}

type IntegerFieldResult struct {
	Count uint `json:"count,omitempty"`
	Min   int  `json:"min"`
	Max   int  `json:"max"`
}

func (k *IntegerFieldResult) HasValues() bool {
	return k.Min < k.Max
}

func (k *IntegerFieldResult) AddValue(value interface{}) {
	v, ok := value.(int)
	if !ok {
		return
	}
	if v < k.Min {
		k.Min = v
	} else if v > k.Max {
		k.Max = v
	}
	k.Count++
}

type DecimalFieldResult struct {
	Count uint    `json:"count,omitempty"`
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
}

func (k *DecimalFieldResult) AddValue(value interface{}) {
	v, ok := value.(float64)
	if !ok {
		return
	}
	if v < k.Min {
		k.Min = v
	} else if v > k.Max {
		k.Max = v
	}
	k.Count++
}

func (k *DecimalFieldResult) HasValues() bool {
	return k.Min < k.Max
}

// func (i *Index) GetFacetsFromResult(ids types.ItemList, filters *Filters, sortIndex *types.SortIndex) []JsonFacet {
// 	l := uint(len(ids))
// 	needsTruncation := l > 16144
// 	fields := make(map[uint]FieldResult)

// 	var base *types.BaseField

// 	for key, field := range i.Facets {
// 		base = field.GetBaseField()
// 		if base.HideFacet || ((base.Type == "" || base.Priority < 1000) && needsTruncation) {

// 		} else {
// 			switch field.GetType() {
// 			case types.FacetKeyType:
// 				fields[key] = &KeyFieldResult{
// 					Values: make(map[string]uint)}
// 			case types.FacetIntegerType:
// 				fields[key] = &IntegerFieldResult{}
// 			case types.FacetNumberType:
// 				fields[key] = &DecimalFieldResult{}
// 			}
// 		}

// 	}

// 	var f FieldResult

// 	var ok bool
// 	var item *types.Item
// 	var field interface{}
// 	var id uint

// 	for id = range ids {
// 		item, ok = i.Items[id]
// 		if !ok {
// 			continue
// 		}
// 		for id, field = range (*item).GetFields() {
// 			if f, ok = fields[id]; ok {
// 				f.AddValue(field)
// 			}
// 		}
// 	}

// 	return i.mapToSlice(fields, sortIndex)
// }

type JsonFacet struct {
	*types.BaseField
	Selected interface{} `json:"selected,omitempty"`
	Result   FieldResult `json:"result,omitempty"`
}

// func (i *Index) mapToSlice(fields map[uint]FieldResult, sortIndex *types.SortIndex) []JsonFacet {
// 	l := min(len(fields), 35)
// 	sorted := make([]JsonFacet, len(fields))
// 	idx := 0
// 	var base *types.BaseField
// 	for _, id := range *sortIndex {
// 		f, ok := fields[id]
// 		if ok {
// 			indexField, baseOk := i.Facets[id]
// 			if !baseOk {
// 				continue
// 			}
// 			base = indexField.GetBaseField()
// 			if !base.HideFacet && f.HasValues() {
// 				sorted[idx] = JsonFacet{
// 					base,
// 					nil,
// 					f,
// 				}
// 				idx++
// 				if idx >= l {
// 					break
// 				}
// 			}
// 		}
// 	}
// 	return sorted[:idx]
// }
