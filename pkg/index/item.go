package index

import (
	"tornberg.me/facet-search/pkg/facet"
)

type EnergyRating struct {
	Value string `json:"value,omitempty"`
	Min   string `json:"min,omitempty"`
	Max   string `json:"max,omitempty"`
}

type ItemProp struct {
	Url string `json:"url"`
	//Tree            []string      `json:"tree"`
	Disclaimer      string       `json:"disclaimer,omitempty"`
	ReleaseDate     string       `json:"releaseDate,omitempty"`
	SaleStatus      string       `json:"saleStatus"`
	MarginPercent   float64      `json:"mp,omitempty"`
	PresaleDate     string       `json:"presaleDate,omitempty"`
	Restock         string       `json:"restock,omitempty"`
	AdvertisingText string       `json:"advertisingText,omitempty"`
	Img             string       `json:"img,omitempty"`
	BadgeUrl        string       `json:"badgeUrl,omitempty"`
	EnergyRating    EnergyRating `json:"energyRating,omitempty"`
	BulletPoints    string       `json:"bp,omitempty"`
	LastUpdate      int64        `json:"lastUpdate,omitempty"`
	Created         int64        `json:"created,omitempty"`
	Buyable         bool         `json:"buyable"`
	BuyableInStore  bool         `json:"buyableInStore"`
}

type LocationStock []struct {
	Id    string `json:"id"`
	Level string `json:"level"`
}

type BaseItem struct {
	ItemProp
	StockLevel string        `json:"stockLevel,omitempty"`
	Stock      LocationStock `json:"stock"`
	Id         uint          `json:"id"`
	Sku        string        `json:"sku"`
	Title      string        `json:"title"`
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
	Fields FieldValues `json:"values"`
}

func MakeResultItem(item *DataItem) ResultItem {
	return ResultItem{
		BaseItem: &item.BaseItem,
		Fields:   item.getFieldValues(),
	}
}

func (item *DataItem) GetPrice() int {

	if item.IntegerFields != nil {
		for _, field := range item.IntegerFields {
			if field.Id == 4 {
				return field.Value
			}
		}
	}
	return 0

}

func (item *DataItem) MergeKeyFields(updates []CategoryUpdate) {
	for _, update := range updates {
		found := false
		for idx, f := range item.Fields {
			if f.Id == update.Id {
				item.Fields[idx].Value = update.Value
				found = true
			}
		}
		if !found {
			item.Fields = append(item.Fields, facet.KeyFieldValue{
				Value: update.Value,
				Id:    update.Id,
			})
		}
	}
}

func ToResultItem(item *DataItem, resultItem *ResultItem) {
	resultItem.BaseItem = &item.BaseItem
	resultItem.Fields = item.getFieldValues()
}

func (item *DataItem) getFieldValues() FieldValues {
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
	return fields
	//}
	//return item.fieldValues

}
