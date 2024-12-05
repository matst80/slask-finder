package index

import (
	"fmt"

	"github.com/matst80/slask-finder/pkg/types"
)

type EnergyRating struct {
	Value string `json:"value,omitempty"`
	Min   string `json:"min,omitempty"`
	Max   string `json:"max,omitempty"`
}

type PriceTuple struct {
	IncVat int `json:"inc"`
	ExVat  int `json:"exl"`
}

type OutletItem struct {
	ArticleNumber string     `json:"sku,opmitempty"`
	Price         PriceTuple `json:"price,omitempty"`
	Title         string     `json:"title"`
}

type ItemProp struct {
	Url string `json:"url"`
	//Tree            []string      `json:"tree"`
	Disclaimer      string        `json:"disclaimer,omitempty"`
	ReleaseDate     string        `json:"releaseDate,omitempty"`
	SaleStatus      string        `json:"saleStatus"`
	MarginPercent   float64       `json:"mp,omitempty"`
	PresaleDate     string        `json:"presaleDate,omitempty"`
	Restock         string        `json:"restock,omitempty"`
	AdvertisingText string        `json:"advertisingText,omitempty"`
	Img             string        `json:"img,omitempty"`
	BadgeUrl        string        `json:"badgeUrl,omitempty"`
	EnergyRating    *EnergyRating `json:"energyRating,omitempty"`
	BulletPoints    string        `json:"bp,omitempty"`
	LastUpdate      int64         `json:"lastUpdate,omitempty"`
	Created         int64         `json:"created,omitempty"`
	Buyable         bool          `json:"buyable"`
	Description     string        `json:"description,omitempty"`
	BuyableInStore  bool          `json:"buyableInStore"`
	BoxSize         string        `json:"boxSize,omitempty"`
	CheapestBItem   *OutletItem   `json:"bItem,omitempty"`
	AItem           *OutletItem   `json:"aItem,omitempty"`
}

type BaseItem struct {
	ItemProp
	StockLevel string              `json:"stockLevel,omitempty"`
	Stock      types.LocationStock `json:"stock"`
	Id         uint                `json:"id"`
	Sku        string              `json:"sku"`
	Title      string              `json:"title"`
}

type DataItem struct {
	*BaseItem
	Taxonomy []string         `json:"taxonomy"`
	Fields   types.ItemFields `json:"values"`
}

func ToItemArray(items []DataItem) []types.Item {
	baseItems := make([]types.Item, 0, len(items))
	for _, item := range items {
		baseItems = append(baseItems, &item)
	}
	return baseItems
}

func (item *DataItem) GetId() uint {
	return item.Id
}

func (item *DataItem) IsDeleted() bool {
	return item.SaleStatus == "MDD"
}

func (item *DataItem) GetPrice() int {
	priceField, ok := item.Fields.GetFacetValue(4)
	if !ok {
		return 0
	}
	return int(getPrice(priceField))
}

func (item *DataItem) GetStock() types.LocationStock {
	return item.Stock
}

func (item *DataItem) GetFields() map[uint]interface{} {
	return item.Fields.GetFacets()
}

func (item *DataItem) GetLastUpdated() int64 {
	return item.LastUpdate
}

func (item *DataItem) GetCreated() int64 {
	return item.Created
}

func getPrice(priceField interface{}) float64 {

	switch priceField := priceField.(type) {
	case int64:
		return float64(priceField)
	case float64:
		return priceField
	}
	return 0
}

func (item *DataItem) GetPopularity() float64 {
	v := 0.0
	price := float64(0)
	orgPrice := float64(0)
	grade := 0
	noGrades := 0

	isOwnBrand := false
	for id, f := range item.Fields.GetFacets() {
		if id == 4 {
			price = getPrice(f)
		}
		if id == 5 {
			orgPrice = getPrice(f)
		}
		if id == 6 {
			grade = int(getPrice(f))
		}
		if id == 7 {
			noGrades = int(getPrice(f))
		}
		if id == 9 {
			if soldby, ok := f.(string); ok {
				if soldby == "Elgiganten" {
					isOwnBrand = true
				}
			}
		}
	}

	if orgPrice > 0 && orgPrice-price > 0 {
		//sdiscount := orgPrice - price
		//v += ((float64(discount) / float64(orgPrice)) * 100000.0) + (float64(discount) / 5.0)
		v += 7500
	}
	if item.Buyable {
		v += 5000
	}
	if price > 99999900 {
		v -= 2500
	}
	if price < 10000 {
		v -= 800
	}
	if len(item.Stock) == 0 && item.StockLevel == "0" {
		v -= 6000
	}
	if item.BadgeUrl != "" {
		v += 4500
	}
	if isOwnBrand {
		v += 4000
		if item.MarginPercent < 99 && item.MarginPercent >= 0 {
			v -= item.MarginPercent * 100
		}
	}
	if item.BadgeUrl != "" {
		v += 2500
	}

	return v + (float64(grade*min(noGrades, 100)) / 10)

}

func (item *DataItem) GetTitle() string {
	return item.Title
}

func (item *DataItem) ToString() string {
	return fmt.Sprintf("%s %s %s", item.Sku, item.Title, item.BulletPoints)
}

func (item *DataItem) GetBaseItem() types.BaseItem {
	return types.BaseItem{
		Id:    item.Id,
		Sku:   item.Sku,
		Title: item.Title,
		Price: item.GetPrice(),
		Img:   item.Img,
	}
}
func (item *DataItem) GetItem() interface{} {
	return item.BaseItem
}

func (item *DataItem) GetStockLevel() string {
	return item.StockLevel
}

func (item *DataItem) MergeKeyFields(updates []types.CategoryUpdate) bool {
	changed := false
	for _, update := range updates {
		field, ok := item.Fields[update.Id]
		if !ok {
			item.Fields[update.Id] = update.Value
			changed = true
		} else if field != update.Value {
			field = update.Value
			changed = true
		}
	}
	return changed
}
