package types

type BaseField struct {
	Id            uint    `json:"id"`
	Name          string  `json:"name"`
	Description   string  `json:"description,omitempty"`
	Priority      float64 `json:"prio,omitempty"`
	Type          string  `json:"valueType,omitempty"`
	LinkedId      uint    `json:"linkedId,omitempty"`
	ValueSorting  uint    `json:"sorting,omitempty"`
	Searchable    bool    `json:"searchable,omitempty"`
	HideFacet     bool    `json:"hide,omitempty"`
	CategoryLevel int     `json:"categoryLevel,omitempty"`
	// IgnoreCategoryIfSearched bool    `json:"-"`
	// IgnoreIfInSearch         bool    `json:"-"`
}

type FacetRequest struct {
	*Filters
	Stock        []string `json:"stock" schema:"stock"`
	Query        string   `json:"query" schema:"query"`
	IgnoreFacets []uint   `json:"skipFacets" schema:"sf"`
}

func (f *FacetRequest) HasField(id uint) bool {
	for _, v := range f.StringFilter {
		if v.Id == id {
			return true
		}
	}
	for _, v := range f.RangeFilter {
		if v.Id == id {
			return true
		}
	}
	return false
}

func (f *FacetRequest) IsIgnored(id uint) bool {
	for _, v := range f.IgnoreFacets {
		if v == id {
			return true
		}
	}
	return false
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
	GetStock() map[string]string
	GetFields() map[uint]interface{}
	IsDeleted() bool
	IsSoftDeleted() bool
	GetPrice() int
	GetDiscount() *int
	GetRating() (int, int)
	GetFieldValue(id uint) (interface{}, bool)
	GetLastUpdated() int64
	GetCreated() int64
	//GetPopularity() float64
	GetTitle() string
	ToString() string
	ToStringList() []string
	GetBaseItem() BaseItem
	MergeKeyFields(updates []CategoryUpdate) bool
	GetItem() interface{}
}

const FacetKeyType = 1
const FacetNumberType = 2
const FacetIntegerType = 3
const FacetTreeType = 4

type Facet interface {
	GetType() uint
	Match(data interface{}) *ItemList
	MatchAsync(data interface{}, results chan<- *ItemList)
	GetBaseField() *BaseField
	AddValueLink(value interface{}, item Item) bool
	RemoveValueLink(value interface{}, id uint)
	GetValues() []interface{}
}
