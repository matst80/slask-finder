package types

import "sync"

type Settings struct {
	mu              sync.RWMutex
	FieldsToIndex   []uint               `json:"fieldsToIndex"`
	FacetRelations  []FacetRelationGroup `json:"facetRelations"`
	PopularityRules *ItemPopularityRules `json:"popularityRules"`
}

type ItemRequirement struct {
	FacetId uint        `json:"facetId"`
	Value   interface{} `json:"value"`
}

type ValueConverter string

const (
	NoConverter = ValueConverter("none")
	ValueToMin  = ValueConverter("valueToMin")
	ValueToMax  = ValueConverter("valueToMax")
)

type FacetRelation struct {
	Name               string `json:"name,omitempty"`
	FacetId            uint   `json:"fromId"`
	DestinationFacetId uint   `json:"toId"`
	//ItemRequirements   []ItemRequirement `json:"requiredForItem"`
	//AdditionalQueries  []ItemRequirement `json:"additionalQueries"`
	ValueConverter ValueConverter `json:"converter"`
}

type FacetRelationGroup struct {
	Name              string            `json:"name"`
	GroupId           int               `json:"groupId"`
	ItemRequirements  []ItemRequirement `json:"requiredForItem"`
	AdditionalQueries []ItemRequirement `json:"additionalQueries"`
	Relations         []FacetRelation   `json:"relations"`
}

const (
	MB_GROUP             = 1
	RAM_GROUP            = 2
	CPU_GROUP            = 3
	LIQUID_COOLING_GROUP = 4
	AIR_COOLING_GROUP    = 5
	PSU_GROUP            = 6
	M2_GROUP             = 7
)

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
	FacetRelations: []FacetRelationGroup{
		// CPU
		{
			Name:    "Passande Vattenkylare",
			GroupId: LIQUID_COOLING_GROUP,
			ItemRequirements: []ItemRequirement{
				{
					FacetId: 32,
					Value:   "PT272",
				},
			},
			AdditionalQueries: []ItemRequirement{
				{
					FacetId: 33,
					Value:   "PT1302",
				},
			},
			Relations: []FacetRelation{
				{
					FacetId:            35990,
					DestinationFacetId: 36307,
					ValueConverter:     ValueToMin,
				},
			},
		},
		{
			Name:    "Passande moderkort",
			GroupId: MB_GROUP,
			ItemRequirements: []ItemRequirement{
				{
					FacetId: 32,
					Value:   "PT272",
				},
			},
			AdditionalQueries: []ItemRequirement{
				{
					FacetId: 32,
					Value:   "PT264",
				},
			},
			Relations: []FacetRelation{
				{
					FacetId:            32103,
					DestinationFacetId: 32103,
					ValueConverter:     NoConverter,
				},
				{
					FacetId:            36202,
					DestinationFacetId: 30276,
					ValueConverter:     NoConverter,
				},
			},
		},
		// RAM
		{
			Name:    "Passande minne",
			GroupId: RAM_GROUP,
			ItemRequirements: []ItemRequirement{
				{
					FacetId: 32,
					Value:   "PT272",
				},
			},
			AdditionalQueries: []ItemRequirement{
				{
					FacetId: 32,
					Value:   "PT269",
				},
			},
			Relations: []FacetRelation{
				{
					FacetId:            35980,
					DestinationFacetId: 31191,
					ValueConverter:     ValueToMin,
				},
			},
		},
		// Motherboard
		{
			Name:    "Passande CPU",
			GroupId: CPU_GROUP,
			ItemRequirements: []ItemRequirement{
				{
					FacetId: 32,
					Value:   "PT264",
				},
			},
			AdditionalQueries: []ItemRequirement{
				{
					FacetId: 32,
					Value:   "PT272",
				},
			},
			Relations: []FacetRelation{
				{
					FacetId:            32103,
					DestinationFacetId: 32103,
					ValueConverter:     NoConverter,
				},
				{
					FacetId:            30276,
					DestinationFacetId: 36202,
					ValueConverter:     NoConverter,
				},
			},
		},
		{
			Name:    "Passande minne",
			GroupId: RAM_GROUP,
			ItemRequirements: []ItemRequirement{
				{
					FacetId: 32,
					Value:   "PT264",
				},
			},
			AdditionalQueries: []ItemRequirement{
				{
					FacetId: 32,
					Value:   "PT269",
				},
			},
			Relations: []FacetRelation{
				{
					FacetId:            35921,
					DestinationFacetId: 35921,
					ValueConverter:     NoConverter,
				},
				{
					FacetId:            30857,
					DestinationFacetId: 30857,
					ValueConverter:     NoConverter,
				},
			},
		},
		{
			Name:    "Passande vattenkylare",
			GroupId: LIQUID_COOLING_GROUP,
			ItemRequirements: []ItemRequirement{
				{
					FacetId: 32,
					Value:   "PT264",
				},
			},
			AdditionalQueries: []ItemRequirement{
				{
					FacetId: 33,
					Value:   "PT1302",
				},
			},
			Relations: []FacetRelation{
				{
					FacetId:            35980,
					DestinationFacetId: 32077,
					ValueConverter:     NoConverter,
				},
			},
		},
		{
			Name:    "Passande luftkylare",
			GroupId: LIQUID_COOLING_GROUP,
			ItemRequirements: []ItemRequirement{
				{
					FacetId: 32,
					Value:   "PT264",
				},
			},
			AdditionalQueries: []ItemRequirement{
				{
					FacetId: 33,
					Value:   "PT1303",
				},
			},
			Relations: []FacetRelation{
				{
					FacetId:            35980,
					DestinationFacetId: 32077,
					ValueConverter:     NoConverter,
				},
			},
		},
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
