package facet

import (
	"testing"
)

func TestValueField_SingleMatch(t *testing.T) {
	f := &KeyField[string]{
		BaseField: &BaseField{
			Name: "test",
		},
		values: map[string]IdList{
			"test": makeIdList(1, 2, 3),
			"hej":  makeIdList(2, 3, 4),
		},
	}

	matching := f.Matches("test")
	if !matchAll(matching.ids, 1, 2, 3) {
		t.Errorf("Expected [1, 2, 3] but got %v", matching)
	}
}

// func TestValueField_MultipleMatches(t *testing.T) {
// 	f := &Field[string]{
// 		Name: "test",

// 		values: map[string]IdList{
// 			"test": makeIdList(1, 2, 3),
// 			"hej":  makeIdList(2, 3, 4),
// 		},
// 	}

// 	matching := f.Matches(func(v string) bool {
// 		return v == "test" || v == "hej"
// 	})
// 	if !reflect.DeepEqual(matching.Ids, []int64{1, 2, 3, 4}) {
// 		t.Errorf("Expected [1, 2, 3, 4] but got %v", matching)
// 	}
// }
