package index

import (
	"encoding/json"
	"github.com/matst80/slask-finder/pkg/types"
	"testing"
)

var item = &types.MockItem{
	Id:    1,
	Title: "Hello",
	Fields: map[uint]interface{}{
		10: "World",
	},
	OrgPrice: 200,
	Buyable:  true,
	Stock:    make([]types.LocationStock, 4),
	Price:    100,
}

func TestNotEmptyRule(t *testing.T) {

	rule := &NotEmptyRule{
		PropertyName: "Title",
		ValueIfMatch: 100,
	}
	rule2 := &NotEmptyRule{
		FieldId:      10,
		ValueIfMatch: 200,
	}
	rule3 := &NotEmptyRule{
		PropertyName: "Kalle",
		ValueIfMatch: 300,
	}
	res := CollectPopularity(item, rule, rule2, rule3)
	if res != 300 {
		t.Errorf("Expected 300 but got %v", res)
	}
}

func TestStringMatchRule_GetValue_Normal(t *testing.T) {
	res := CollectPopularity(item, &MatchRule{
		Match:           "World",
		Invert:          false,
		FieldId:         10,
		ValueIfMatch:    100,
		ValueIfNotMatch: -100,
	})
	if res != 100 {
		t.Errorf("Expected 100 but got %v", res)
	}
}

func TestStringMatchRule_GetValue_Inverted(t *testing.T) {
	res := CollectPopularity(item, &MatchRule{
		Match:           "World",
		Invert:          true,
		FieldId:         10,
		ValueIfMatch:    100,
		ValueIfNotMatch: -100,
	})
	if res != -100 {
		t.Errorf("Expected -100 but got %v", res)
	}
}

func TestBoolMatchRule_GetValue_Inverted(t *testing.T) {
	res := CollectPopularity(item, &MatchRule{
		Match:           true,
		Invert:          false,
		PropertyName:    "Buyable",
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
		Multiplier:   0.1,
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
	rules := []ItemPopularityRule{
		&MatchRule{
			Match:           "Elgiganten",
			FieldId:         9,
			ValueIfNotMatch: -12000,
		},
		&MatchRule{
			Match:           "Outlet",
			FieldId:         10,
			ValueIfNotMatch: -6000,
		},
		&DiscountRule{
			Multiplier:   30,
			ValueIfMatch: 4500,
		},
		&MatchRule{
			Match:           true,
			PropertyName:    "Buyable",
			ValueIfMatch:    5000,
			ValueIfNotMatch: -2000,
		},
		&OutOfStockRule{
			NoStoreMultiplier: 20,
			NoStockValue:      -6000,
		},
		&MatchRule{
			Match:           "",
			PropertyName:    "BadgeUrl",
			Invert:          true,
			ValueIfNotMatch: 4500,
		},
		&NumberLimitRule{
			Limit:           99999900,
			Comparator:      ">",
			ValueIfMatch:    -2500,
			ValueIfNotMatch: 0,
			FieldId:         4,
		},
		&NumberLimitRule{
			Limit:           10000,
			Comparator:      "<",
			ValueIfMatch:    -800,
			ValueIfNotMatch: 0,
			FieldId:         4,
		},
		&PercentMultiplierRule{
			Multiplier:   50,
			Min:          0,
			Max:          100,
			PropertyName: "MarginPercent",
		},
		&RatingRule{
			Multiplier:     0.06,
			SubtractValue:  -20,
			ValueIfNoMatch: 0,
		},
		&AgedRule{
			HourMultiplier: -0.019,
			PropertyName:   "Created",
		},
		&AgedRule{
			HourMultiplier: -0.0002,
			PropertyName:   "LastUpdate",
		},
	}
	res := CollectPopularity(item, rules...)
	if res > -18000 {
		t.Errorf("Expected -12000 but got %v", res)
	}
	jsonString, err := json.Marshal(rules)
	if err != nil {
		t.Error(err)
	}
	t.Log(string(jsonString))
	rules2 := []ItemPopularityRule{}
	err = json.Unmarshal(jsonString, &rules2)
	if err != nil {
		t.Error(err)
	}

}
