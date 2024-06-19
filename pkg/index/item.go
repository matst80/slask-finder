package index

import (
	"tornberg.me/facet-search/pkg/facet"
)

type EnergyRating struct {
	Value string `json:"value"`
	Min   string `json:"min"`
	Max   string `json:"max"`
}

type ItemProp struct {
	Url             string        `json:"url"`
	Tree            []string      `json:"tree"`
	ReleaseDate     string        `json:"releaseDate,omitempty"`
	SaleStatus      string        `json:"saleStatus"`
	PresaleDate     string        `json:"presaleDate,omitempty"`
	Restock         string        `json:"restock,omitempty"`
	AdvertisingText string        `json:"advertisingText,omitempty"`
	Img             string        `json:"img,omitempty"`
	BadgeUrl        string        `json:"badgeUrl,omitempty"`
	EnergyRating    *EnergyRating `json:"energyRating,omitempty"`
	BulletPoints    []string      `json:"bp,omitempty"`
}

type BaseItem struct {
	Id    uint     `json:"id"`
	Sku   string   `json:"sku"`
	Title string   `json:"title"`
	Props ItemProp `json:"props"`
}

type DataItem struct {
	BaseItem
	facet.ItemFields
	//fieldValues   *FieldValues
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
	//if item.fieldValues == nil {

	fields := FieldValues{}
	if item.Fields != nil {
		for _, value := range item.Fields {
			fields[value.Id] = value.Value
		}
	}
	if item.DecimalFields != nil {
		for _, value := range item.DecimalFields {
			fields[value.Id] = value.Value
		}
	}
	if item.IntegerFields != nil {
		for _, value := range item.IntegerFields {
			fields[value.Id] = value.Value
		}
	}
	return &fields
	//}
	//return item.fieldValues

}
