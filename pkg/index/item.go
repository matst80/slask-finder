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

type BaseItem struct {
	ItemProp
	StockLevel string              `json:"stockLevel,omitempty"`
	Stock      facet.LocationStock `json:"stock"`
	Id         uint                `json:"id"`
	Sku        string              `json:"sku"`
	Title      string              `json:"title"`
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

func (item DataItem) GetId() uint {
	return item.Id
}

// func (item *DataItem) GetPrice() int {

// 	if item.Fields != nil {
// 		for id, field := range item.Fields {
// 			if id == 4 {
// 				return field.(int)
// 			}
// 		}
// 	}
// 	return 0

// }

// func (item *DataItem) MergeKeyFields(updates []CategoryUpdate) {
// 	for _, update := range updates {
// 		item.Fields[update.Id] = update.Value
// 	}
// }

func (item DataItem) IsDeleted() bool {
	return item.SaleStatus == "MDD"
}

func (item DataItem) GetPrice() int {
	return item.Fields[4].(int)
}

func (item DataItem) GetStock() facet.LocationStock {
	return item.Stock
}

func ToResultItem(item *DataItem, resultItem *ResultItem) {
	resultItem.BaseItem = &item.BaseItem
	resultItem.Fields = item.getFieldValues()
}

func (item *DataItem) getFieldValues() FieldValues {
	return item.Fields

}

func (item DataItem) GetFields() map[uint]interface{} {
	return item.ItemFields.Fields
}

func (item DataItem) GetLastUpdated() int64 {
	return item.LastUpdate
}

func (item DataItem) GetCreated() int64 {
	return item.Created
}

func (item DataItem) GetPopularity() float64 {
	v := 0.0
	price := 0
	orgPrice := 0
	grade := 0
	noGrades := 0

	for id, f := range item.Fields {
		if id == 4 {
			price = f.(int)
		}
		if id == 5 {
			orgPrice = f.(int)
		}
		if id == 6 {
			grade = f.(int)
		}
		if id == 7 {
			noGrades = f.(int)
		}
	}

	if orgPrice > 0 && orgPrice-price > 0 {
		discount := orgPrice - price
		v += ((float64(discount) / float64(orgPrice)) * 100000.0) + (float64(discount) / 5.0)
	}
	if item.Buyable || item.BuyableInStore {
		v += 5000
	}
	if price > 99999900 {
		v -= 2500
	}
	if price < 10000 {
		v -= 800
	}
	if price%900 == 0 {
		v += 700
	}
	v += item.MarginPercent * 400
	return v + float64(grade*min(noGrades, 500))

}

func (item DataItem) GetTitle() string {
	return item.Title
}

func (item DataItem) ToString() string {
	return item.Title
}

func (item DataItem) GetBaseItem() facet.BaseItem {
	return facet.BaseItem{
		Id:    item.Id,
		Sku:   item.Sku,
		Title: item.Title,
		Price: item.GetPrice(),
		Img:   item.Img,
	}
}
