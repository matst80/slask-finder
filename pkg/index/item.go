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
	fieldValues   *FieldValues
}

type ItemKeyField struct {
	Value *string `json:"value"`
}

type ItemNumberField[K facet.FieldNumberValue] struct {
	Value K `json:"value"`
}

// type Item struct {
// 	*BaseItem
// 	Fields        map[uint]ItemKeyField
// 	DecimalFields map[uint]ItemNumberField[float64]
// 	IntegerFields map[uint]ItemNumberField[int]
// 	fieldValues   *FieldValues
// }

type FieldValues map[uint]interface{}

type ResultItem struct {
	*BaseItem
	Fields *FieldValues `json:"values"`
}

func (item *DataItem) getFieldValues() *FieldValues {
	if item.fieldValues == nil {

		fields := FieldValues{}
		for key, value := range item.Fields {
			fields[key] = value
		}
		for key, value := range item.DecimalFields {
			fields[key] = value
		}
		for key, value := range item.IntegerFields {
			fields[key] = value
		}
		item.fieldValues = &fields
	}
	return item.fieldValues

}
