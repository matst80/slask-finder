package facet

import (
	"reflect"
	"testing"

	"tornberg.me/facet-search/pkg/types"
)

func TestPresortedSorting(t *testing.T) {
	sortIndex := types.SortIndex{
		1,
		3,
		2,
		4,
		5,
	}
	ids := map[uint]struct{}{4: {}, 2: {}, 1: {}, 3: {}}

	sortedIds := sortIndex.SortMap(ids, 10)

	expected := []uint{1, 3, 2, 4}
	if !reflect.DeepEqual(sortedIds, expected) {
		t.Errorf("Expected %v but got %v", expected, sortedIds)
	}
}
