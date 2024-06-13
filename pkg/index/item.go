package index

import (
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
	Value *string `json:"value"`
}

type ItemNumberField[K facet.FieldNumberValue] struct {
	Value K `json:"value"`
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
