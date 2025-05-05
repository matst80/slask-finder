package types

import (
	"log"
	"slices"
	"sync"
)

type Settings struct {
	mu               sync.RWMutex
	SearchMergeLimit int                  `json:"searchMergeLimit"`
	SuggestFacets    []uint               `json:"suggestFacets"`
	ProductTypeId    uint                 `json:"productTypeId"`
	FieldsToIndex    []uint               `json:"fieldsToIndex"`
	FacetRelations   []FacetRelationGroup `json:"facetRelations"`
	PopularityRules  *ItemPopularityRules `json:"popularityRules"`
	FacetGroups      []FacetGroup         `json:"facetGroups"`
}

type FacetGroup struct {
	Id          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type ItemRequirement struct {
	FacetId uint        `json:"facetId"`
	Exclude bool        `json:"exclude,omitempty"`
	Value   interface{} `json:"value"`
}

type ValueConverter string

const (
	NoConverter = ValueConverter("none")
	ValueToMin  = ValueConverter("valueToMin")
	ValueToMax  = ValueConverter("valueToMax")
)

type FacetRelation struct {
	Name               string         `json:"name,omitempty"`
	FacetId            uint           `json:"fromId"`
	DestinationFacetId uint           `json:"toId"`
	ValueConverter     ValueConverter `json:"converter"`
}

type FacetRelationGroup struct {
	Name              string            `json:"name"`
	GroupId           int               `json:"groupId"`
	ItemRequirements  []ItemRequirement `json:"requiredForItem"`
	AdditionalQueries []ItemRequirement `json:"additionalQueries"`
	Relations         []FacetRelation   `json:"relations"`
}

func (f *FacetRelationGroup) GetFilter(item Item) []StringFilter {
	result := make([]StringFilter, 0)
	for _, additionalQuery := range f.AdditionalQueries {
		keyValue, ok := AsKeyFilterValue(additionalQuery.Value)
		if !ok {
			log.Printf("Failed to convert %v to key filter value", additionalQuery.Value)
			continue
		}
		result = append(result, StringFilter{
			Id:    additionalQuery.FacetId,
			Not:   additionalQuery.Exclude,
			Value: keyValue,
		})
	}
	for _, relation := range f.Relations {
		itemValue, ok := item.GetFieldValue(relation.FacetId)
		if !ok {
			continue
		}
		keyValue, ok := AsKeyFilterValue(itemValue)
		if !ok {
			log.Printf("Failed to convert %v to key filter value", itemValue)
			continue
		}
		if relation.ValueConverter == NoConverter {
			result = append(result, StringFilter{
				Id:    relation.DestinationFacetId,
				Value: keyValue,
			})
		}
	}
	return result
}

func AsStringArray(value interface{}) []string {
	itemValues := []string{}
	switch input := value.(type) {
	case []string:
		for _, item := range input {
			itemValues = append(itemValues, item)
		}
	case []interface{}:
		for _, item := range input {
			if v, ok := item.(string); ok {
				itemValues = append(itemValues, v)
			}
		}
	case string:
		itemValues = append(itemValues, input)
	}
	return itemValues
}

func matchInterfaceValues(value interface{}, matchValue interface{}) bool {
	if value == nil {
		return false
	}
	if matchValue == nil {

		return true
	}

	itemValues := AsStringArray(value)

	matchItems := AsStringArray(matchValue)

	for _, item := range itemValues {
		if slices.Contains(matchItems, item) {
			return true
		}
	}
	return false

}

func (f *FacetRelationGroup) Matches(item Item) bool {
	for _, relation := range f.ItemRequirements {
		itemValue, ok := item.GetFieldValue(relation.FacetId)
		if !ok {
			log.Printf("Item %d does not have field %d", item.GetId(), relation.FacetId)
			return false
		}
		matches := matchInterfaceValues(itemValue, relation.Value)
		if relation.Exclude {
			return !matches
		}
		return matches
	}
	for _, relation := range f.Relations {
		_, ok := item.GetFieldValue(relation.FacetId)
		if !ok {
			log.Printf("Item %d does not have related field %d", item.GetId(), relation.FacetId)
			return false
		}
	}
	return true
}

var CurrentSettings = &Settings{
	mu:               sync.RWMutex{},
	FacetGroups:      []FacetGroup{},
	ProductTypeId:    31158,
	SearchMergeLimit: 10,
	FieldsToIndex: []uint{
		2,
		31158,
		//12,
		//13,
		30290,
		//11,
		10,
	},
	SuggestFacets: []uint{
		2,
		31158,
		30290,
		10,
		11,
	},
	FacetRelations: []FacetRelationGroup{},
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
		// &PercentMultiplierRule{
		// 	Multiplier: 50,
		// 	Min:        0,
		// 	Max:        100,
		// 	RuleSource: RuleSource{
		// 		Source:       Property,
		// 		PropertyName: "MarginPercent",
		// 	},
		// },
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
