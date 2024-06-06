package facet

import (
	"reflect"
	"testing"
)

func TestValueField_SingleMatch(t *testing.T) {
	f := &ValueField{
		Field: Field{
			Name: "test",
		},
		values: map[string][]int64{
			"test": {1, 2, 3},
			"hej":  {2, 3, 4},
		},
	}

	matching := f.Matches("test")
	if !reflect.DeepEqual(matching.Ids, []int64{1, 2, 3}) {
		t.Errorf("Expected [1, 2, 3] but got %v", matching)
	}
}

func TestValueField_MultipleMatches(t *testing.T) {
	f := &ValueField{
		Field: Field{
			Name: "test",
		},
		values: map[string][]int64{
			"test": {1, 2, 3},
			"hej":  {2, 3, 4},
		},
		// values: []StringValue{
		// 	{
		// 		Value: "test",
		// 		ids:   []int64{1, 2, 3},
		// 	},
		// 	{
		// 		Value: "hej",
		// 		ids:   []int64{2, 3, 4},
		// 	},
		// },
	}

	matching := f.Matches("test", "hej")
	if !reflect.DeepEqual(matching.Ids, []int64{1, 2, 3, 4}) {
		t.Errorf("Expected [1, 2, 3, 4] but got %v", matching)
	}
}
