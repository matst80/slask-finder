package index

import (
	"encoding/json"
	"fmt"
	"io"
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
	Price         PriceTuple `json:"price"`
	Title         string     `json:"title"`
}

type MarginPercent float64

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
	Id         uint  `json:"id"`
	LastUpdate int64 `json:"lastUpdate,omitempty"`
	Created    int64 `json:"created,omitempty"`

	Disclaimer       string `json:"disclaimer,omitempty"`
	ReleaseDate      string `json:"releaseDate,omitempty"`
	SaleStatus       string `json:"saleStatus"`
	OnlineSaleStatus string `json:"onlineSaleStatus"`
	PresaleDate      string `json:"presaleDate,omitempty"`
	Restock          string `json:"restock,omitempty"`
	AdvertisingText  string `json:"advertisingText,omitempty"`
	Img              string `json:"img,omitempty"`
	BadgeUrl         string `json:"badgeUrl,omitempty"`
	BulletPoints     string `json:"bp,omitempty"`

	Description string `json:"description,omitempty"`

	BoxSize       string        `json:"boxSize,omitempty"`
	ArticleType   string        `json:"articleType,omitempty"`
	CheapestBItem *OutletItem   `json:"bItem,omitempty"`
	AItem         *OutletItem   `json:"aItem,omitempty"`
	EnergyRating  *EnergyRating `json:"energyRating,omitempty"`
	MarginPercent MarginPercent `json:"mp,omitempty"`

	Stock          map[string]string `json:"stock"`
	Sku            string            `json:"sku"`
	Title          string            `json:"title"`
	Buyable        bool              `json:"buyable"`
	BuyableInStore bool              `json:"buyableInStore"`
}

type DataItem struct {
	*BaseItem
	//Taxonomy []string         `json:"taxonomy"`
	Fields types.ItemFields `json:"values"`
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
	v, ok := item.GetNumberFieldValue(3)
	return ok && v > 0
}

func (item *DataItem) GetPropertyValue(name string) any {
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
	p, ok := item.GetNumberFieldValue(4)
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

	priceField, ok := item.GetNumberFieldValue(4)
	if !ok {
		return 0
	}
	return getNumberValue[int](priceField)
}

func (item *DataItem) GetStock() map[string]string {
	return item.Stock
}

// func (item *DataItem) GetFields() map[uint]interface{} {
// 	return item.Fields.GetFacets()
// }

func (m *DataItem) GetStringFields() map[uint]string {
	return m.Fields.GetStringFields()
	// ret := make(map[uint]string, len(m.Fields))
	// for k, v := range m.Fields {
	// 	switch value := v.(type) {
	// 	case string:
	// 		ret[k] = value
	// 	case []string:
	// 		ret[k] = strings.Join(value, ";;")
	// 	case []any:
	// 		strs := make([]string, 0, len(value))
	// 		for _, iv := range value {
	// 			if s, ok := iv.(string); ok {
	// 				strs = append(strs, s)
	// 			}
	// 		}
	// 		if len(strs) > 0 {
	// 			ret[k] = strings.Join(strs, ";;")
	// 		}
	// 	}
	// }
	// return ret
}

func (m *DataItem) GetNumberFields() map[uint]float64 {
	return m.Fields.GetNumberFields()
	// ret := make(map[uint]float64, len(m.Fields))
	// for k, v := range m.Fields {
	// 	switch value := v.(type) {
	// 	case int:
	// 		ret[k] = float64(value)
	// 	case int64:
	// 		ret[k] = float64(value)
	// 	case float64:
	// 		ret[k] = value
	// 	case []any:
	// 		if len(value) == 1 {
	// 			switch nvalue := value[0].(type) {
	// 			case int:
	// 				ret[k] = float64(nvalue)
	// 			case int64:
	// 				ret[k] = float64(nvalue)
	// 			case float64:
	// 				ret[k] = nvalue
	// 			}
	// 		}
	// 	}
	// }
	// return ret
}

func (m *DataItem) GetStringFieldValue(id uint) (string, bool) {
	return m.Fields.GetStringFieldValue(id)
	// fields := m.GetStringFields()
	// v, ok := fields[id]
	// if ok && len(v) > 0 {
	// 	return v, true
	// }

	// return "", false
}

func (m *DataItem) GetStringsFieldValue(id uint) ([]string, bool) {
	return m.Fields.GetStringsFieldValue(id)
	//  fields := m.GetStringFields()
	// if v, ok := fields[id]; ok {
	// 	return strings.Split(v, ";;"), true
	// }

	// return nil, false
}
func (m *DataItem) GetNumberFieldValue(id uint) (float64, bool) {
	return m.Fields.GetNumberFieldValue(id)
	// fields := m.GetNumberFields()
	// if v, ok := fields[id]; ok {
	// 	return v, true
	// }
	// return 0, false
}

// func (item *DataItem) GetFacetValue(id uint) (any, bool) {
// 	return item.Fields.GetFacetValue(id)
// }

// func (item *DataItem) GetFacets() map[uint]any {
// 	return item.Fields.GetFacets()
// }

func (item *DataItem) GetRating() (int, int) {
	average, ok := item.GetNumberFieldValue(6)
	if !ok {
		return 0, 0
	}
	grades, ok := item.GetNumberFieldValue(7)
	if !ok {
		return 0, 0
	}
	return getNumberValue[int](average), getNumberValue[int](grades)
}

func (item *DataItem) CanHaveEmbeddings() bool {
	outlet, okA := item.Fields.GetStringFieldValue(10)
	seller, okB := item.Fields.GetStringFieldValue(9)
	if !okA || !okB {
		return false
	}
	return outlet != "Outlet" && (seller == "Elgiganten" || seller == "Elkjøp" || seller == "Gigantti")
}
func (item *DataItem) GetEmbeddingsText() (string, error) {
	return item.Title + "\n" + item.BulletPoints, nil
}

func (item *DataItem) GetLastUpdated() int64 {
	return item.LastUpdate
}

func (item *DataItem) GetCreated() int64 {
	return item.Created
}

func getNumberValue[K float64 | int](fieldValue any) K {

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
	orgPriceValue, ok := item.Fields.GetNumberFieldValue(5)
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

func (item *DataItem) GetTitle() string {
	return item.Title
}

func getStringValues(fieldValue any, found bool) []string {
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
	fieldValues = append(fieldValues, item.BulletPoints)
	for _, id := range types.CurrentSettings.FieldsToIndex {
		v, found := item.GetStringsFieldValue(id)
		if found {
			fieldValues = append(fieldValues, v...)
		}
	}

	return fieldValues
}

func (item *DataItem) ToString() string {
	return strings.Join(item.ToStringList(), " ")
}

func (item *DataItem) Write(w io.Writer) (int, error) {
	bytes, err := json.Marshal(item)
	if err != nil {
		return 0, err
	}

	b, err := w.Write(bytes)
	if err != nil {
		return b, err
	}
	n, err := w.Write([]byte("\n"))
	return b + n, err
}

//	func (item *DataItem) GetBaseItem() types.BaseItem {
//		return types.BaseItem{
//			Id:    item.Id,
//			Sku:   item.Sku,
//			Title: item.Title,
//			Price: item.GetPrice(),
//			Img:   item.Img,
//		}
//	}
// func (item *DataItem) GetItem() interface{} {
// 	return item.BaseItem
// }
