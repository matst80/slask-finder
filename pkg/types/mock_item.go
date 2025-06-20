package types

type MockItem struct {
	Id          uint
	Sku         string
	Fields      map[uint]interface{}
	Deleted     bool
	Price       int
	OrgPrice    int
	StockLevel  string
	Stock       map[string]string
	Buyable     bool
	LastUpdated int64
	Created     int64
	Popularity  float64
	Title       string
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

func (m *MockItem) GetPropertyValue(name string) interface{} {
	return nil
}

func (m *MockItem) UpdateBasePopularity(rules ItemPopularityRules) {
	m.Popularity = 1
}

func (m *MockItem) GetRating() (int, int) {
	return 20, 5
}

func (m *MockItem) GetFieldValue(id uint) (interface{}, bool) {
	v, ok := m.Fields[id]
	return v, ok
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

func (m *MockItem) GetStock() map[string]string {
	return m.Stock
}

func (m *MockItem) GetFields() map[uint]interface{} {
	return m.Fields
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

func (m *MockItem) GetItem() interface{} {
	return m
}

type MockField struct {
	Key   uint
	Value interface{}
}

func MakeMockItem(id uint, fields ...MockField) Item {
	ret := &MockItem{
		Id:     id,
		Fields: make(map[uint]interface{}),
	}
	for _, field := range fields {
		ret.Fields[field.Key] = field.Value

	}
	return ret
}

func (m *MockItem) ToItem() Item {
	return m
}
