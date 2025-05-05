package types

type BaseField struct {
	Id               uint    `json:"id"`
	Name             string  `json:"name"`
	Description      string  `json:"description,omitempty"`
	Priority         float64 `json:"prio,omitempty"`
	Type             string  `json:"valueType,omitempty"`
	LinkedId         uint    `json:"linkedId,omitempty"`
	ValueSorting     uint    `json:"sorting,omitempty"`
	Searchable       bool    `json:"searchable,omitempty"`
	HideFacet        bool    `json:"hide,omitempty"`
	CategoryLevel    int     `json:"categoryLevel,omitempty"`
	GroupId          uint    `json:"groupId,omitempty"`
	KeySpecification bool    `json:"isKey,omitempty"`
	InternalOnly     bool    `json:"internal,omitempty"`
	// IgnoreCategoryIfSearched bool    `json:"-"`
	// IgnoreIfInSearch         bool    `json:"-"`
}

type FacetRequest struct {
	*Filters
	Stock        []string `json:"stock" schema:"stock"`
	Query        string   `json:"query" schema:"query"`
	IgnoreFacets []uint   `json:"skipFacets" schema:"sf"`
}

func (s *FacetRequest) Sanitize() {
	if (len(s.StringFilter) > 0 || len(s.RangeFilter) > 0) && s.Query == "*" {
		s.Query = ""
	}
}

func (b *BaseField) UpdateFrom(field *BaseField) {
	if field == nil {
		return
	}
	//b.Id = field.Id
	if field.Name != "" {
		b.Name = field.Name
	}
	if field.Description != "" {
		b.Description = field.Description
	}

	b.Priority = field.Priority
	if field.Type != "" {
		b.Type = field.Type
	}
	b.LinkedId = field.LinkedId
	b.ValueSorting = field.ValueSorting
	b.Searchable = field.Searchable
	b.HideFacet = field.HideFacet
	b.CategoryLevel = field.CategoryLevel
	b.GroupId = field.GroupId
	b.KeySpecification = field.KeySpecification
	b.InternalOnly = field.InternalOnly
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
	// should be config
	if id >= 11 && id <= 14 {
		for _, sf := range f.StringFilter {
			if sf.Id > 9 && sf.Id < 14 && sf.Id != id {
				return false
			}
		}
		return true
	}
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
	GetSku() string
	GetStock() map[string]string
	GetStockLevel() string
	GetFields() map[uint]interface{}
	IsDeleted() bool
	IsSoftDeleted() bool
	GetPropertyValue(name string) interface{}
	GetPrice() int
	GetDiscount() int
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
	GetBasePopularity() float64
	UpdateBasePopularity(rules ItemPopularityRules)
	GetItem() interface{}
}

const FacetKeyType = 1
const FacetNumberType = 2
const FacetIntegerType = 3
const FacetTreeType = 4

type Facet interface {
	GetType() uint
	Match(data interface{}) *ItemList
	// MatchAsync(data interface{}, results chan<- *ItemList)
	GetBaseField() *BaseField
	AddValueLink(value interface{}, id uint) bool
	RemoveValueLink(value interface{}, id uint)
	UpdateBaseField(data *BaseField)
	GetValues() []interface{}
}

type FieldChangeAction = string

const (
	ADD_FIELD    FieldChangeAction = "add"
	REMOVE_FIELD FieldChangeAction = "remove"
	UPDATE_FIELD FieldChangeAction = "update"
)

type FieldChange struct {
	*BaseField
	Action    FieldChangeAction `json:"action"`
	FieldType uint              `json:"fieldType"`
}

func (s *Settings) Lock() {
	s.mu.Lock()
}
func (s *Settings) Unlock() {
	s.mu.Unlock()
}
func (s *Settings) RLock() {
	s.mu.RLock()
}
func (s *Settings) RUnlock() {
	s.mu.RUnlock()
}
