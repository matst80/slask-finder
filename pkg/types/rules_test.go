package types

import (
	"encoding/json"
	"testing"
)

var item = &MockItem{
	Id:    1,
	Title: "Hello",
	Fields: map[uint]interface{}{
		10: "World",
	},
	OrgPrice: 200,
	Buyable:  true,
	Stock:    make([]LocationStock, 4),
	Price:    100,
}

func TestNotEmptyRule(t *testing.T) {

	rule := &NotEmptyRule{
		RuleSource: RuleSource{
			Source:       Property,
			PropertyName: "Title",
		},
		ValueIfMatch: 100,
	}
	rule2 := &NotEmptyRule{
		RuleSource: RuleSource{
			FieldId: 10,
		},
		ValueIfMatch: 200,
	}
	rule3 := &NotEmptyRule{
		RuleSource: RuleSource{
			PropertyName: "Kalle",
		},
		ValueIfMatch: 300,
	}
	res := CollectPopularity(item, rule, rule2, rule3)
	if res != 300 {
		t.Errorf("Expected 300 but got %v", res)
	}
}

func TestStringMatchRule_GetValue_Normal(t *testing.T) {
	res := CollectPopularity(item, &MatchRule{
		Match:  "World",
		Invert: false,
		RuleSource: RuleSource{
			FieldId: 10,
		},
		ValueIfMatch:    100,
		ValueIfNotMatch: -100,
	})
	if res != 100 {
		t.Errorf("Expected 100 but got %v", res)
	}
}

// [{"match":"Elgiganten","fieldId":9,"value":0,"valueIfNotMatch":-12000,"$type":"MatchRule"},{"match":"Outlet","fieldId":10,"value":0,"valueIfNotMatch":-6000,"$type":"MatchRule"},{"multiplier":30,"valueIfMatch":4500,"$type":"DiscountRule"},{"match":true,"property":"Buyable","value":5000,"valueIfNotMatch":-2000,"$type":"MatchRule"},{"noStoreMultiplier":20,"noStockValue":-6000,"$type":"MatchRule"},{"match":"","invert":true,"property":"BadgeUrl","value":0,"valueIfNotMatch":4500,"$type":"MatchRule"},{"limit":99999900,"comparator":"\u003e","value":-2500,"valueIfNotMatch":0,"fieldId":4,"$type":"NumberLimitRule"},{"limit":10000,"comparator":"\u003c","value":-800,"valueIfNotMatch":0,"fieldId":4,"$type":"NumberLimitRule"},{"multiplier":50,"min":0,"max":100,"property":"MarginPercent","$type":"NumberLimitRule"},{"multiplier":0.06,"subtractValue":-20,"valueIfNoMatch":0,"$type":"RatingRule"},{"hourMultiplier":-0.019,"property":"Created","$type":"AgedRule"},{"hourMultiplier":-0.0002,"property":"LastUpdate","$type":"AgedRule"}]
func TestStringMatchRule_GetValue_Inverted(t *testing.T) {
	res := CollectPopularity(item, &MatchRule{
		Match:  "World",
		Invert: true,
		RuleSource: RuleSource{
			FieldId: 10,
		},
		ValueIfMatch:    100,
		ValueIfNotMatch: -100,
	})
	if res != -100 {
		t.Errorf("Expected -100 but got %v", res)
	}
}

func TestBoolMatchRule_GetValue_Inverted(t *testing.T) {
	res := CollectPopularity(item, &MatchRule{
		Match:  true,
		Invert: false,
		RuleSource: RuleSource{
			PropertyName: "Buyable",
		},
		ValueIfMatch:    100,
		ValueIfNotMatch: -100,
	})
	if res != 100 {
		t.Errorf("Expected 100 but got %v", res)
	}
}

func TestOutOfStockRule_GetValue(t *testing.T) {
	res := CollectPopularity(item, &OutOfStockRule{
		NoStoreMultiplier: 2,
		NoStockValue:      -100,
	})
	if res != -100 {
		t.Errorf("Expected -100 but got %v", res)
	}
}

func TestDiscountRule_GetValue(t *testing.T) {
	res := CollectPopularity(item, &DiscountRule{
		Multiplier:   10,
		ValueIfMatch: 100,
	})
	if res != 110 {
		t.Errorf("Expected 110 but got %v", res)
	}
}

func TestRatingRule_GetValue(t *testing.T) {
	res := CollectPopularity(item, &RatingRule{
		Multiplier:     0.06,
		SubtractValue:  -20,
		ValueIfNoMatch: 0,
	})
	if res != 2.4 {
		t.Errorf("Expected 2.4 but got %v", res)
	}
}

func TestRecreateRules(t *testing.T) {
	rules := JsonTypes{
		&MatchRule{
			Match: "Elgiganten",
			RuleSource: RuleSource{
				Source:  FieldId,
				FieldId: 9,
			},
			ValueIfNotMatch: -12000,
		},
		&MatchRule{
			Match: "Outlet",
			RuleSource: RuleSource{
				Source:  FieldId,
				FieldId: 10,
			},
			ValueIfNotMatch: -6000,
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
			Invert:          true,
			ValueIfNotMatch: 4500,
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
		&AgedRule{
			HourMultiplier: -0.0019,
			RuleSource: RuleSource{
				Source:       Property,
				PropertyName: "Created",
			},
		},
		&AgedRule{
			HourMultiplier: -0.00002,
			RuleSource: RuleSource{
				Source:       Property,
				PropertyName: "LastUpdate",
			},
		},
	}
	res := CollectPopularity(item, FromJsonTypes[ItemPopularityRule](rules)...)

	jsonString, err := json.Marshal(rules)
	if err != nil {
		t.Error(err)
	}
	t.Log(string(jsonString))
	rules2 := JsonTypes{}
	err = json.Unmarshal(jsonString, &rules2)
	if err != nil {
		t.Error(err)
	}
	res2 := CollectPopularity(item, FromJsonTypes[ItemPopularityRule](rules)...)
	if diff(res2, res) > 0.001 {
		t.Errorf("Expected %f but got %f", res, res2)
	}

}

func diff(a, b float64) float64 {
	if a > b {
		return a - b
	}
	return b - a
}
