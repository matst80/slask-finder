package index

import (
	"sort"

	"tornberg.me/facet-search/pkg/facet"
)

type ItemProp interface{}

type Item struct {
	Id           int64               `json:"id"`
	Sku          string              `json:"sku"`
	Title        string              `json:"title"`
	Props        map[string]ItemProp `json:"props"`
	Fields       map[int64]string    `json:"values"`
	NumberFields map[int64]float64   `json:"numberValues"`
	BoolFields   map[int64]bool      `json:"boolValues"`
}

type Sort struct {
	FieldId int64 `json:"fieldId"`
	Asc     bool  `json:"asc"`
}

func MakeSortFromNumberField(items map[int64]Item, fieldId int64) facet.SortIndex {
	l := len(items)
	sortIndex := make(facet.SortIndex, l)
	sortMap := make(facet.ByValue, l)
	for idx, item := range items {
		sortMap[idx] = facet.Lookup{Id: item.Id, Value: item.NumberFields[fieldId]}
	}
	sort.Sort(sortMap)
	for idx, item := range sortMap {
		sortIndex[idx] = item.Id
	}
	return sortIndex
}
