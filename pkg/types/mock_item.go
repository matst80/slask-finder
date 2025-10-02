package types

import (
	"encoding/json"
	"io"
	"log"
	"strings"
)

type MockItem struct {
	Id  uint
	Sku string
	//Fields      map[uint]interface{}
	StringFields map[uint]string
	NumberFields map[uint]float64
	Deleted      bool
	Price        int
	OrgPrice     int
	StockLevel   string
	Stock        map[string]uint
	Buyable      bool
	LastUpdated  int64
	Created      int64
	Popularity   float64
	Title        string
}

func (m *MockItem) GetDiscount() int {

	if m.OrgPrice > m.Price {
		return m.OrgPrice - m.Price
	}
	return 0
}

func (m *MockItem) HasStock() bool {
	return true
}

func (m *MockItem) GetBasePopularity() float64 {
	return m.Popularity
}

func (m *MockItem) GetPropertyValue(name string) any {
	return nil
}

func (m *MockItem) GetRating() (int, int) {
	return 20, 5
}

// func (m *MockItem) GetFieldValue(id uint) (interface{}, bool) {
// 	v, ok := m.Fields[id]
// 	return v, ok
// }

func (m *MockItem) GetStringFields() map[uint]string {
	return m.StringFields
}

func (m *MockItem) GetNumberFields() map[uint]float64 {
	return m.NumberFields
}

func (m *MockItem) GetStringFieldValue(id uint) (string, bool) {
	if v, ok := m.StringFields[id]; ok && len(v) > 0 {
		return v, true
	}
	return "", false
}

func (m *MockItem) GetStringsFieldValue(id uint) ([]string, bool) {
	if v, ok := m.StringFields[id]; ok {
		return strings.Split(v, ";"), true
	}
	return nil, false
}
func (m *MockItem) GetNumberFieldValue(id uint) (float64, bool) {
	if v, ok := m.NumberFields[id]; ok {
		return v, true
	}
	return 0, false
}

func (m *MockItem) GetId() uint {
	return m.Id
}

func (m *MockItem) GetSku() string {
	return m.Sku
}

func (m *MockItem) IsSoftDeleted() bool {
	return false
}

func (m *MockItem) GetStock() map[string]uint {
	return m.Stock
}

func (m *MockItem) IsDeleted() bool {
	return m.Deleted
}

func (m *MockItem) GetPrice() int {
	return m.Price
}

func (m *MockItem) GetLastUpdated() int64 {
	return m.LastUpdated
}

func (m *MockItem) GetCreated() int64 {
	return m.Created
}

func (m *MockItem) GetPopularity() float64 {
	return m.Popularity
}

func (m *MockItem) GetTitle() string {
	return m.Title
}

func (m *MockItem) ToString() string {
	return m.Title
}

func (m *MockItem) ToStringList() []string {
	return []string{m.Title}
}

func (m *MockItem) CanHaveEmbeddings() bool {
	return true
}
func (m *MockItem) GetEmbeddingsText() (string, error) {
	return m.Title, nil
}

func (m *MockItem) GetBaseItem() BaseItem {
	return BaseItem{
		Id:    m.Id,
		Sku:   "",
		Title: m.Title,
		Price: m.Price,
		Img:   "",
	}
}

func (item *MockItem) MergeKeyFields(updates []CategoryUpdate) bool {
	return false
}

func (m *MockItem) GetItem() any {
	return m
}

func (m *MockItem) GetFields() []uint {
	fields := make([]uint, 0, len(m.StringFields)+len(m.NumberFields))
	for k := range m.StringFields {
		fields = append(fields, k)
	}
	for k := range m.NumberFields {
		fields = append(fields, k)
	}
	return fields
}

type MockField struct {
	Key   uint
	Value any
}

func MakeMockItem(id uint, fields ...MockField) Item {
	ret := &MockItem{
		Id:           id,
		StringFields: make(map[uint]string),
		NumberFields: make(map[uint]float64),
		Deleted:      false,
		Stock:        make(map[string]uint),
	}
	for _, field := range fields {
		switch v := field.Value.(type) {
		case string:
			ret.StringFields[field.Key] = v
		case []string:
			ret.StringFields[field.Key] = strings.Join(v, ";")
		case []any:
			strs := make([]string, 0, len(v))
			for _, vi := range v {
				if s, ok := vi.(string); ok {
					strs = append(strs, s)
				} else {
					log.Printf("Non-string value in string array for id %d: %T", field.Key, vi)
				}
			}
			if len(strs) > 0 {
				ret.StringFields[field.Key] = strings.Join(strs, ";")
			}
		case float64:
			ret.NumberFields[field.Key] = v
		case int:
			ret.NumberFields[field.Key] = float64(v)
		case int64:
			ret.NumberFields[field.Key] = float64(v)
		default:
			log.Printf("Unsupported field type for id %d: %T", field.Key, v)
		}

	}
	return ret
}

func (m *MockItem) ToItem() Item {
	return m
}

func (m *MockItem) Write(w io.Writer) (int, error) {
	bytes, err := json.Marshal(m)
	if err != nil {
		return 0, err
	}
	return w.Write(bytes)
}
