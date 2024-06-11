package index

import (
	"math"
	"sort"

	"tornberg.me/facet-search/pkg/facet"
)

type ItemProp interface{}

type BaseItem struct {
	Id    int64               `json:"id"`
	Sku   string              `json:"sku"`
	Title string              `json:"title"`
	Props map[string]ItemProp `json:"props"`
}

type DataItem struct {
	BaseItem
	Fields        map[int64]string  `json:"values"`
	DecimalFields map[int64]float64 `json:"numberValues"`
	IntegerFields map[int64]int     `json:"integerValues"`
	BoolFields    map[int64]bool    `json:"boolValues"`
}

type ItemKeyField[K facet.FieldKeyValue] struct {
	*facet.KeyField[K]
	Value K `json:"value"`
}

type ItemNumberField[K facet.FieldNumberValue] struct {
	*facet.NumberField[K]
	Value K `json:"value"`
}

type Item struct {
	BaseItem
	Fields        map[int64]ItemKeyField[string]
	DecimalFields map[int64]ItemNumberField[float64]
	IntegerFields map[int64]ItemNumberField[int]
	BoolFields    map[int64]ItemKeyField[bool]
}

// type Sort struct {
// 	FieldId int64 `json:"fieldId"`
// 	Asc     bool  `json:"asc"`
// }

func MakeSortFromDecimalField(items map[int64]Item, fieldId int64) facet.SortIndex {
	l := len(items)
	sortIndex := make(facet.SortIndex, l)
	sortMap := make(facet.ByValue, l)
	for idx, item := range items {
		sortMap[idx] = facet.Lookup{Id: item.Id, Value: math.Abs(item.DecimalFields[fieldId].Value - 3000)}
	}
	sort.Sort(sortMap)
	for idx, item := range sortMap {
		sortIndex[idx] = item.Id
	}
	return sortIndex
}
