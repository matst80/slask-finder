package index

import (
	"encoding/json"
	"fmt"
	"strings"

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
	ArticleNumber string     `json:"sku,omitempty"`
	Price         PriceTuple `json:"price,omitempty"`
	Title         string     `json:"title"`
}

type MarginPercent float64

type ItemProp struct {
	Url string `json:"url"`

	Disclaimer       string        `json:"disclaimer,omitempty"`
	ReleaseDate      string        `json:"releaseDate,omitempty"`
	SaleStatus       string        `json:"saleStatus"`
	OnlineSaleStatus string        `json:"onlineSaleStatus"`
	PresaleDate      string        `json:"presaleDate,omitempty"`
	Restock          string        `json:"restock,omitempty"`
	AdvertisingText  string        `json:"advertisingText,omitempty"`
	Img              string        `json:"img,omitempty"`
	BadgeUrl         string        `json:"badgeUrl,omitempty"`
	BulletPoints     string        `json:"bp,omitempty"`
	LastUpdate       int64         `json:"lastUpdate,omitempty"`
	Created          int64         `json:"created,omitempty"`
	Buyable          bool          `json:"buyable"`
	Description      string        `json:"description,omitempty"`
	BuyableInStore   bool          `json:"buyableInStore"`
	BoxSize          string        `json:"boxSize,omitempty"`
	ArticleType      string        `json:"articleType,omitempty"`
	CheapestBItem    *OutletItem   `json:"bItem,omitempty"`
	AItem            *OutletItem   `json:"aItem,omitempty"`
	EnergyRating     *EnergyRating `json:"energyRating,omitempty"`
	MarginPercent    MarginPercent `json:"mp,omitempty"`
}

var AllowConditionalData = false

func (a *MarginPercent) UnmarshalJSON(b []byte) error {
	var v float64
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	*a = MarginPercent(v)
	return nil
}

func (a MarginPercent) MarshalJSON() ([]byte, error) {
	if AllowConditionalData {
		return json.Marshal(float64(a))
	}
	return json.Marshal(0.0)
}

type BaseItem struct {
	ItemProp
	//StockLevel string            `json:"stockLevel,omitempty"`
	Stock     map[string]string `json:"stock"`
	Sku       string            `json:"sku"`
	Title     string            `json:"title"`
	Id        uint              `json:"id"`
	baseScore float64
}

type DataItem struct {
	*BaseItem
	//Taxonomy []string         `json:"taxonomy"`
	Fields types.ItemFields `json:"values"`
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

func (item *DataItem) GetSku() string {
	return item.Sku
}

func (item *DataItem) IsDeleted() bool {
	softDeleted := item.IsSoftDeleted()

	if softDeleted {
		return true
	}
	if item.SaleStatus == "999" {
		return true
	}
	return item.SaleStatus == "MDD"
}

func (item *DataItem) HasStock() bool {
	v, ok := item.GetFieldValue(3)
	return ok && v != nil
}

func (item *DataItem) GetPropertyValue(name string) interface{} {
	switch name {
	case "Sku":
		return item.Sku
	case "Title":
		return item.Title
	case "BadgeUrl":
		return item.BadgeUrl
	case "Img":
		return item.Img
	//case "StockLevel":
	//	return item.StockLevel
	case "Stock":
		return item.Stock
	case "Buyable":
		return item.Buyable
	case "BuyableInStore":
		return item.BuyableInStore
	case "BoxSize":
		return item.BoxSize
	case "EnergyRating":
		return item.EnergyRating
	case "BulletPoints":
		return item.BulletPoints
	case "Description":
		return item.Description
	case "ReleaseDate":
		return item.ReleaseDate
	case "SaleStatus":
		return item.SaleStatus
	case "OnlineSaleStatus":
		return item.OnlineSaleStatus
	case "MarginPercent":
		return item.MarginPercent
	case "Restock":
		return item.Restock
	case "PresaleDate":
		return item.PresaleDate
	case "AdvertisingText":
		return item.AdvertisingText
	case "Created":
		return item.Created
	case "LastUpdate":
		return item.LastUpdate
	case "CheapestBItem":
		return item.CheapestBItem
	case "AItem":
		return item.AItem
	case "ArticleType":
		return item.ArticleType
	default:
		return nil
	}
}

func (item *DataItem) IsSoftDeleted() bool {
	p, ok := item.Fields.GetFacetValue(4)
	if !ok {
		return true
	}
	price := getNumberValue[int](p)
	if price <= 200 {
		return true
	}
	if !item.Buyable {
		return true
	}
	if price > 99999000 && price <= 100000000 {
		return true
	}
	if item.SaleStatus == "DIS" {
		return true
	}
	if item.SaleStatus == "DIO" {
		return true
	}
	return false
}

func (item *DataItem) GetPrice() int {

	priceField, ok := item.Fields.GetFacetValue(4)
	if !ok {
		return 0
	}
	return getNumberValue[int](priceField)
}

func (item *DataItem) GetStock() map[string]string {
	return item.Stock
}

func (item *DataItem) GetFields() map[uint]interface{} {
	return item.Fields.GetFacets()
}

func (item *DataItem) GetFieldValue(id uint) (interface{}, bool) {
	v, ok := item.Fields[id]
	return v, ok
}

func (item *DataItem) GetRating() (int, int) {
	average, ok := item.GetFieldValue(6)
	if !ok {
		return 0, 0
	}
	grades, ok := item.GetFieldValue(7)
	if !ok {
		return 0, 0
	}
	return getNumberValue[int](average), getNumberValue[int](grades)
}

func (item *DataItem) GetLastUpdated() int64 {
	return item.LastUpdate
}

func (item *DataItem) GetCreated() int64 {
	return item.Created
}

func getNumberValue[K float64 | int](fieldValue interface{}) K {

	switch value := fieldValue.(type) {
	case int:
		return K(value)
	case int64:
		return K(value)
	case float64:
		return K(value)
	}
	return 0
}

func (item *DataItem) GetDiscount() int {
	price := item.GetPrice()
	orgPriceValue, ok := item.GetFieldValue(5)
	if !ok {
		return 0
	}
	orgPrice := getNumberValue[int](orgPriceValue)
	if orgPrice > 0 && orgPrice > price {
		discount := orgPrice - price
		return discount
	}
	return 0
}

// func (item *DataItem) GetPopularity() float64 {
// 	v := 0.0
// 	price := float64(0)
// 	orgPrice := float64(0)
// 	grade := 0
// 	noGrades := 0
// 	isOutlet := false
// 	isOwnBrand := false
// 	for id, f := range item.Fields.GetFacets() {
// 		if id == 4 {
// 			price = getNumberValue[float64](f)
// 		}
// 		if id == 5 {
// 			orgPrice = getNumberValue[float64](f)
// 		}
// 		if id == 6 {
// 			grade = getNumberValue[int](f)
// 		}
// 		if id == 7 {
// 			noGrades = getNumberValue[int](f)
// 		}
// 		if id == 9 {
// 			if soldby, ok := f.(string); ok {
// 				if soldby == "Elgiganten" {
// 					isOwnBrand = true
// 				}
// 			}
// 		}
// 		if id == 10 {
// 			if cat, ok := f.(string); ok {
// 				if cat == "Outlet" {
// 					isOutlet = true
// 				}
// 			}
// 		}
// 	}

// 	if isOutlet {
// 		v -= 6000
// 	}
// 	if orgPrice > 0 && orgPrice-price > 0 {
// 		//sdiscount := orgPrice - price
// 		//v += ((float64(discount) / float64(orgPrice)) * 100000.0) + (float64(discount) / 5.0)
// 		v += 7500
// 	}
// 	if item.Buyable {
// 		v += 5000
// 	}
// 	if price > 99999900 {
// 		v -= 2500
// 	}
// 	if price < 10000 {
// 		v -= 800
// 	}
// 	if len(item.Stock) == 0 && item.StockLevel == "0" {
// 		v -= 6000
// 	}
// 	if item.BadgeUrl != "" {
// 		v += 4500
// 	}
// 	if !isOwnBrand {
// 		v -= 12000
// 	}
// 	if item.MarginPercent < 99 && item.MarginPercent >= 0 {
// 		v -= item.MarginPercent * 50
// 	}
// 	if item.BadgeUrl != "" {
// 		v += 2500
// 	}
// 	ageInDays := (time.Now().UnixNano() - item.Created) / (60 * 60 * 24 * 1000)
// 	if ageInDays < 5 {
// 		v += 2500
// 	}
// 	v -= float64(ageInDays) / 2.0
// 	if grade == 0 && noGrades == 0 {
// 		return v
// 	}
// 	return v + (float64((grade-20)*noGrades) / 5)

// }

func (item *DataItem) GetTitle() string {
	return item.Title
}

func getStringValues(fieldValue interface{}, found bool) []string {
	if !found {
		return []string{}
	}
	switch value := fieldValue.(type) {
	case []string:
		return value
	case string:
		return []string{value}
	case int:
		return []string{fmt.Sprintf("%d", value)}
	case int64:
		return []string{fmt.Sprintf("%d", value)}
	case float64:
		return []string{fmt.Sprintf("%f", value)}
	default:
		return []string{}
	}
}

func (item *DataItem) ToStringList() []string {
	fieldValues := make([]string, 0)
	fieldValues = append(fieldValues, item.Title)
	fieldValues = append(fieldValues, item.Sku)

	for _, id := range types.CurrentSettings.FieldsToIndex {
		fieldValues = append(fieldValues, getStringValues(item.GetFieldValue(id))...)
	}

	return fieldValues
}

func (item *DataItem) ToString() string {
	return strings.Join(item.ToStringList(), " ")
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

func (item *DataItem) GetBasePopularity() float64 {
	return item.baseScore
}

func (item *DataItem) UpdateBasePopularity(rules types.ItemPopularityRules) {
	item.baseScore = types.CollectPopularity(item, rules...)
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
