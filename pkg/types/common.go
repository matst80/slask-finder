package types

type BaseField struct {
	Id               uint    `json:"id"`
	Name             string  `json:"name"`
	Description      string  `json:"description,omitempty"`
	Priority         float64 `json:"prio,omitempty"`
	Type             string  `json:"type,omitempty"`
	LinkedId         uint    `json:"linkedId,omitempty"`
	ValueSorting     uint    `json:"sorting,omitempty"`
	HideFacet        bool    `json:"-"`
	CategoryLevel    int     `json:"categoryLevel,omitempty"`
	IgnoreIfInSearch bool    `json:"-"`
}

type LocationStock []struct {
	Id    string `json:"id"`
	Level string `json:"level"`
}

type BaseItem struct {
	Id    uint
	Sku   string
	Title string
	Price int
	Img   string
}

type CategoryUpdate struct {
	Id    uint   `json:"id"`
	Value string `json:"value"`
}

type Item interface {
	GetId() uint
	GetStock() LocationStock
	GetFields() map[uint]interface{}
	IsDeleted() bool
	IsSoftDeleted() bool
	GetPrice() int
	GetLastUpdated() int64
	GetCreated() int64
	GetPopularity() float64
	GetTitle() string
	ToString() string
	GetBaseItem() BaseItem
	MergeKeyFields(updates []CategoryUpdate) bool
	GetItem() interface{}
}

const FacetKeyType = 1
const FacetNumberType = 2
const FacetIntegerType = 3

type Facet interface {
	GetType() uint
	Match(data interface{}) *ItemList
	GetBaseField() *BaseField
	AddValueLink(value interface{}, item Item) bool
	RemoveValueLink(value interface{}, id uint)
	GetValues() []interface{}
	Size() int
}
