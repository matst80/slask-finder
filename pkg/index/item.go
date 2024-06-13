package index

import (
	"math"
	"sort"

	"tornberg.me/facet-search/pkg/facet"
)

type ItemProp interface{}

type BaseItem struct {
	Id    uint                `json:"id"`
	Sku   string              `json:"sku"`
	Title string              `json:"title"`
	Props map[string]ItemProp `json:"props"`
}

type DataItem struct {
	BaseItem
	Fields        map[uint]string  `json:"values"`
	DecimalFields map[uint]float64 `json:"numberValues"`
	IntegerFields map[uint]int     `json:"integerValues"`
	BoolFields    map[uint]bool    `json:"boolValues"`
}

type ItemKeyField struct {
	Value     *string `json:"value"`
	ValueHash uint    `json:"-"`
}

type ItemNumberField[K facet.FieldNumberValue] struct {
	Value  K   `json:"value"`
	Bucket int `json:"-"`
}

type Item struct {
	BaseItem
	Fields        map[uint]ItemKeyField
	DecimalFields map[uint]ItemNumberField[float64]
	IntegerFields map[uint]ItemNumberField[int]
}

type ResultItem struct {
	BaseItem
	Fields map[uint]interface{} `json:"values"`
}

// type Sort struct {
// 	FieldId int `json:"fieldId"`
// 	Asc     bool  `json:"asc"`
// }

func MakeSortFromDecimalField(items map[uint]Item, fieldId uint) facet.SortIndex {
	l := len(items)
	sortIndex := make(facet.SortIndex, l)
	sortMap := make(facet.ByValue, l)
	for idx, item := range items {
		sortMap[idx] = facet.Lookup{Id: item.Id, Value: math.Abs(item.DecimalFields[fieldId].Value - float64(300000))}
	}
	sort.Sort(sortMap)
	for idx, item := range sortMap {
		sortIndex[idx] = item.Id
	}
	return sortIndex
}
