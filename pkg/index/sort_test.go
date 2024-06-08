package index

import (
	"reflect"
	"testing"
)

func TestPresortedSorting(t *testing.T) {
	sortIndex := SortIndex{
		1,
		2,
		3,
		4,
		5,
	}
	ids := []int64{4, 2, 1, 2}

	sortedIds := sortIndex.SortIds(ids)
	// if sortedIds[0] != 1 || sortedIds[1] != 2 || sortedIds[2] != 4 {
	// 	t.Errorf("Expected [1, 2, 4] but got %v", sortedIds)
	// }
	expected := []int64{1, 2, 2, 4}
	if !reflect.DeepEqual(sortedIds, expected) {
		t.Errorf("Expected %v but got %v", expected, sortedIds)
	}
}
