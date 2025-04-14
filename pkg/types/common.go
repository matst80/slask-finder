package types

import "sync"

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

type FieldChangeAction = string

const (
	ADD_FIELD    FieldChangeAction = "add"
	REMOVE_FIELD FieldChangeAction = "remove"
	UPDATE_FIELD FieldChangeAction = "update"
)

type FieldChange struct {
	Action    FieldChangeAction `json:"action"`
	FieldType uint              `json:"fieldType"`
	*BaseField
}

type Settings struct {
	mu              sync.RWMutex
	FieldsToIndex   []uint
	PopularityRules *ItemPopularityRules
}

var CurrentSettings = &Settings{
	mu: sync.RWMutex{},
	FieldsToIndex: []uint{
		2,
		31158,
		//12,
		//13,
		30290,
		//11,
		//10,
	},
	PopularityRules: &ItemPopularityRules{
		&MatchRule{
			Match: "Elgiganten",
			RuleSource: RuleSource{
				Source:  FieldId,
				FieldId: 9,
			},
			ValueIfNotMatch: -12000,
		},
		&MatchRule{
			Match: "Apple",
			RuleSource: RuleSource{
				Source:  FieldId,
				FieldId: 2,
			},
			ValueIfMatch: 2500,
		},
		&MatchRule{
			Match: "Samsung",
			RuleSource: RuleSource{
				Source:  FieldId,
				FieldId: 2,
			},
			ValueIfMatch: 2300,
		},
		&MatchRule{
			Match: "Google",
			RuleSource: RuleSource{
				Source:  FieldId,
				FieldId: 2,
			},
			ValueIfMatch: 2100,
		},
		&MatchRule{
			Match: "PRE",
			RuleSource: RuleSource{
				Source:       Property,
				PropertyName: "SaleStatus",
			},
			ValueIfMatch: 1500,
		},
		&MatchRule{
			Match: "ZBAN",
			RuleSource: RuleSource{
				Source:  FieldId,
				FieldId: 36,
			},
			ValueIfMatch: -4500,
		},
		&MatchRule{
			Match: "ZMAR",
			RuleSource: RuleSource{
				Source:  FieldId,
				FieldId: 36,
			},
			ValueIfMatch: -4300,
		},
		&MatchRule{
			Match: "Nothing",
			RuleSource: RuleSource{
				Source:  FieldId,
				FieldId: 2,
			},
			ValueIfMatch: 2100,
		},
		&MatchRule{
			Match: "Outlet",
			RuleSource: RuleSource{
				Source:  FieldId,
				FieldId: 10,
			},
			ValueIfMatch: -6000,
		},
		&DiscountRule{
			Multiplier:   30,
			ValueIfMatch: 4500,
		},
		&MatchRule{
			Match: true,
			RuleSource: RuleSource{
				Source:       Property,
				PropertyName: "Buyable",
			},
			ValueIfMatch:    5000,
			ValueIfNotMatch: -2000,
		},
		&OutOfStockRule{
			NoStoreMultiplier: 20,
			NoStockValue:      -6000,
		},
		&MatchRule{
			Match: "",
			RuleSource: RuleSource{
				Source:       Property,
				PropertyName: "BadgeUrl",
			},
			Invert:          false,
			ValueIfNotMatch: 3500,
		},
		&MatchRule{
			Match: "",
			RuleSource: RuleSource{
				Source:  FieldId,
				FieldId: 21,
			},
			Invert:          false,
			ValueIfNotMatch: 4200,
		},
		&NumberLimitRule{
			Limit:           99999900,
			Comparator:      ">",
			ValueIfMatch:    -2500,
			ValueIfNotMatch: 0,
			RuleSource: RuleSource{
				Source:  FieldId,
				FieldId: 4,
			},
		},
		&NumberLimitRule{
			Limit:           10000,
			Comparator:      "<",
			ValueIfMatch:    -800,
			ValueIfNotMatch: 0,
			RuleSource: RuleSource{
				Source:  FieldId,
				FieldId: 4,
			},
		},
		&PercentMultiplierRule{
			Multiplier: 50,
			Min:        0,
			Max:        100,
			RuleSource: RuleSource{
				Source:       Property,
				PropertyName: "MarginPercent",
			},
		},
		&RatingRule{
			Multiplier:     0.06,
			SubtractValue:  -20,
			ValueIfNoMatch: 0,
		},
		// &MatchRule{
		// 	Match: false,
		// 	RuleSource: RuleSource{
		// 		Source:       Property,
		// 		PropertyName: "Buyable",
		// 	},
		// 	ValueIfMatch:    -200000,
		// 	ValueIfNotMatch: 1000,
		// },
		// &AgedRule{
		// 	HourMultiplier: -0.0019,
		// 	RuleSource: RuleSource{
		// 		Source:       Property,
		// 		PropertyName: "Created",
		// 	},
		// },
		// &AgedRule{
		// 	HourMultiplier: -0.00002,
		// 	RuleSource: RuleSource{
		// 		Source:       Property,
		// 		PropertyName: "LastUpdate",
		// 	},
		// },
	},
}

func (s *Settings) GetPopularityRules() *ItemPopularityRules {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.PopularityRules
}
func (s *Settings) SetPopularityRules(rules *ItemPopularityRules) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.PopularityRules = rules
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
