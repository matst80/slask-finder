package facet

import "github.com/matst80/slask-finder/pkg/types"

type Facets map[uint]FieldResult

type FieldResult interface {
	HasValues() bool
}

type KeyFieldResult struct {
	Values map[string]int `json:"values,omitempty"`
}

func (k *KeyFieldResult) HasValues() bool {
	return len(k.Values) > 1
}

type JsonFacet struct {
	*types.BaseField
	Selected interface{} `json:"selected,omitempty"`
	Result   FieldResult `json:"result,omitempty"`
}
