package index

import (
	"log"
	"strings"

	"github.com/matst80/slask-finder/pkg/types"
)

type StorageDataItem struct {
	*BaseItem
	StringFields map[uint][]string `json:"keys"`
	NumberFields map[uint]float64  `json:"numbers"`
}

// ToInternalDataItem converts a DataItem to an InternalDataItem.
func ToStorageDataItem(dataItem *DataItem) *StorageDataItem {
	return &StorageDataItem{
		BaseItem:     dataItem.BaseItem,
		StringFields: dataItem.GetStringFields(),
		NumberFields: dataItem.GetNumberFields(),
	}
}

func FromStorageDataItem(dataItem *StorageDataItem) DataItem {
	i := DataItem{
		BaseItem: dataItem.BaseItem,
		Fields:   make(types.ItemFields),
	}
	for id, value := range dataItem.StringFields {
		if len(value) == 1 {
			i.Fields[id] = value[0]
		} else if len(value) > 1 {
			i.Fields[id] = value
		}
	}
	for id, value := range dataItem.NumberFields {
		i.Fields[id] = value
	}
	return i
}

func (item *StorageDataItem) GetId() uint {
	return item.Id
}

func (item *StorageDataItem) GetSku() string {
	return item.Sku
}

func (item *StorageDataItem) IsDeleted() bool {
	softDeleted := item.IsSoftDeleted()

	if softDeleted {
		return true
	}
	if item.SaleStatus == "999" {
		return true
	}
	return item.SaleStatus == "MDD"
}

func (item *StorageDataItem) HasStock() bool {
	a, ok := item.GetStringFieldValue(3)
	return ok && (a != "" && a != "0")
}

func (item *StorageDataItem) GetPropertyValue(name string) interface{} {
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

func (item *StorageDataItem) IsSoftDeleted() bool {
	p, ok := item.NumberFields[4]
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

func (item *StorageDataItem) GetPrice() int {

	priceField, ok := item.NumberFields[4]
	if !ok {
		return 0
	}
	return getNumberValue[int](priceField)
}

func (item *StorageDataItem) GetStock() map[string]string {
	return item.Stock
}

// func (item *StorageDataItem) GetFields() map[uint]interface{} {
// 	ret := make([uint])
// 	for id, v := range item.StringFields {
// 		if len(v) == 1 {
// 			ret[id] = v[0]
// 		} else if len(v) > 1 {
// 			ret[id] = v
// 		}
// 	}
// 	for id, v := range item.NumberFields {
// 		ret[id] = v
// 	}
// 	return ret
// }

func (m *StorageDataItem) GetFields() []uint {
	fields := make([]uint, 0, len(m.StringFields)+len(m.NumberFields))
	for k := range m.StringFields {
		fields = append(fields, k)
	}
	for k := range m.NumberFields {
		fields = append(fields, k)
	}
	return fields
}

func (m *StorageDataItem) GetStringFields() map[uint][]string {
	return m.StringFields
}

func (m *StorageDataItem) GetNumberFields() map[uint]float64 {
	return m.NumberFields
}

func (m *StorageDataItem) GetStringFieldValue(id uint) (string, bool) {
	if v, ok := m.StringFields[id]; ok && len(v) > 0 {
		return v[0], true
	}
	return "", false
}

func (m *StorageDataItem) GetStringsFieldValue(id uint) ([]string, bool) {
	if v, ok := m.StringFields[id]; ok {
		return v, true
	}
	return nil, false
}
func (m *StorageDataItem) GetNumberFieldValue(id uint) (float64, bool) {
	if v, ok := m.NumberFields[id]; ok {
		return v, true
	}
	return 0, false
}

func (item *StorageDataItem) GetFieldValue(id uint) (interface{}, bool) {
	if v, ok := item.StringFields[id]; ok {
		return v, true
	}
	if v, ok := item.NumberFields[id]; ok {
		return v, true
	}
	return nil, false
}

// func (item *StorageDataItem) GetFacetValue(id uint) (interface{}, bool) {
// 	return item.Fields.GetFacetValue(id)
// }

// func (item *StorageDataItem) GetFacets() map[uint]interface{} {
// 	return item.Fields.GetFacets()
// }

func (item *StorageDataItem) SetValue(id uint, value interface{}) {
	switch v := value.(type) {
	case string:
		item.StringFields[id] = []string{v}
	case []string:
		item.StringFields[id] = v
	case []interface{}:
		strs := make([]string, 0, len(v))
		for _, vi := range v {
			if s, ok := vi.(string); ok {
				strs = append(strs, s)
			} else {
				log.Printf("Non-string value in string array for id %d: %T", id, vi)
			}
		}
		if len(strs) > 0 {
			item.StringFields[id] = strs
		}
	case float64:
		item.NumberFields[id] = v
	case int:
		item.NumberFields[id] = float64(v)
	case int64:
		item.NumberFields[id] = float64(v)
	case nil:
		// skip
	default:
		log.Printf("Unknown field type for id %d: %T", id, v)
	}
	//item.Fields.SetValue(id, value)
}

func (item *StorageDataItem) GetRating() (int, int) {
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

func (item *StorageDataItem) CanHaveEmbeddings() bool {
	mainCategory, okA := item.StringFields[10]
	seller, okB := item.StringFields[9]
	if !okA || !okB {
		return false
	}
	if len(mainCategory) == 0 || len(seller) == 0 {
		return false
	}

	return mainCategory[0] != "Outlet" && (seller[0] == "Elgiganten" || seller[0] == "Elkjøp" || seller[0] == "Gigantti")
}
func (item *StorageDataItem) GetEmbeddingsText() (string, error) {
	return item.Title + "\n" + item.BulletPoints, nil
}

func (item *StorageDataItem) GetLastUpdated() int64 {
	return item.LastUpdate
}

func (item *StorageDataItem) GetCreated() int64 {
	return item.Created
}

func (item *StorageDataItem) GetDiscount() int {
	price := item.GetPrice()
	orgPriceValue, ok := item.GetNumberFieldValue(5)
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

func (item *StorageDataItem) GetTitle() string {
	return item.Title
}

func (item *StorageDataItem) ToStringList() []string {
	fieldValues := make([]string, 0)
	fieldValues = append(fieldValues, item.Title)
	fieldValues = append(fieldValues, item.Sku)
	fieldValues = append(fieldValues, item.BulletPoints)
	for _, id := range types.CurrentSettings.FieldsToIndex {
		fieldValues = append(fieldValues, getStringValues(item.GetFieldValue(id))...)
	}

	return fieldValues
}

func (item *StorageDataItem) ToString() string {
	return strings.Join(item.ToStringList(), " ")
}
