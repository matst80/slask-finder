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

type StorageItem struct {
	BaseItem
	DataItemFields
}

type DataItemFields struct {
	Fields        []KeyFieldValue     `json:"values"`
	DecimalFields []DecimalFieldValue `json:"numberValues"`
	IntegerFields []IntegerFieldValue `json:"integerValues"`
}

type KeyFieldValue struct {
	Value string `json:"value"`
	Id    uint   `json:"id"`
}

type DecimalFieldValue struct {
	Value float64 `json:"value"`
	Id    uint    `json:"id"`
}

type IntegerFieldValue struct {
	Value int  `json:"value"`
	Id    uint `json:"id"`
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
		for id, value := range item.Fields {
			fields[id] = value
		}
	}
	if item.DecimalFields != nil {
		for id, value := range item.DecimalFields {
			fields[id] = value
		}
	}
	if item.IntegerFields != nil {
		for id, value := range item.IntegerFields {
			fields[id] = value
		}
	}
	return &fields
	//}
	//return item.fieldValues

}
