package index

import (
	"math"
	"sort"

	"tornberg.me/facet-search/pkg/facet"
)

type ItemProp interface{}

type BaseItem struct {
	Id    int                 `json:"id"`
	Sku   string              `json:"sku"`
	Title string              `json:"title"`
	Props map[string]ItemProp `json:"props"`
}

type DataItem struct {
	BaseItem
	Fields        map[int]string  `json:"values"`
	DecimalFields map[int]float64 `json:"numberValues"`
	IntegerFields map[int]int     `json:"integerValues"`
	BoolFields    map[int]bool    `json:"boolValues"`
}

type ItemKeyField struct {
	field     *facet.KeyField
	Value     string `json:"value"`
	ValueHash uint32 `json:"valueHash"`
}

type ItemNumberField[K facet.FieldNumberValue] struct {
	field *facet.NumberField[K]
	Value K `json:"value"`
}

type Item struct {
	BaseItem
	Fields        map[int]ItemKeyField
	DecimalFields map[int]ItemNumberField[float64]
	IntegerFields map[int]ItemNumberField[int]
}

type ResultItem struct {
	BaseItem
	Fields map[int]interface{} `json:"values"`
}

// type Sort struct {
// 	FieldId int `json:"fieldId"`
// 	Asc     bool  `json:"asc"`
// }

func MakeSortFromDecimalField(items map[int]Item, fieldId int) facet.SortIndex {
	l := len(items)
	sortIndex := make(facet.SortIndex, l)
	sortMap := make(facet.ByValue, l)
	for idx, item := range items {
		sortMap[idx] = facet.Lookup{Id: item.Id, Value: math.Abs(item.DecimalFields[fieldId].Value - float64(3000))}
	}
	sort.Sort(sortMap)
	for idx, item := range sortMap {
		sortIndex[idx] = item.Id
	}
	return sortIndex
}
