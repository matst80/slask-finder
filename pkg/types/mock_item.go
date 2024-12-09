package types

type MockItem struct {
	Id            uint
	LocationStock LocationStock
	Fields        map[uint]interface{}
	Deleted       bool
	Price         int
	LastUpdated   int64
	Created       int64
	Popularity    float64
	Title         string
}

func (m *MockItem) GetId() uint {
	return m.Id
}

func (m *MockItem) IsSoftDeleted() bool {
	return false
}

func (m *MockItem) GetStock() LocationStock {
	return m.LocationStock
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

func MakeMockItem(id uint) Item {
	return &MockItem{
		Id: id,
	}
}

func (m *MockItem) ToItem() Item {
	return m
}
