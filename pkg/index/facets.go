package index

import (
	"github.com/matst80/slask-finder/pkg/types"
)

type Facets map[uint]FieldResult

type FieldResult interface {
	//AddValue(value interface{})
	HasValues() bool
}

type KeyFieldResult struct {
	Values map[string]int `json:"values,omitempty"`
}

// func (k *KeyFieldResult) AddValue(input interface{}) {
// 	value, ok := input.(string)
// 	if !ok || value == "" {
// 		return
// 	}
// 	k.Values[value]++
// }

func (k *KeyFieldResult) HasValues() bool {
	return len(k.Values) > 1
}

// func (k *IntegerFieldResult) AddValue(value interface{}) {
// 	v, ok := value.(int)
// 	if !ok {
// 		return
// 	}
// 	if v < k.Min {
// 		k.Min = v
// 	} else if v > k.Max {
// 		k.Max = v
// 	}
// 	k.Count++
// }

type JsonFacet struct {
	*types.BaseField
	Selected interface{} `json:"selected,omitempty"`
	Result   FieldResult `json:"result,omitempty"`
}
