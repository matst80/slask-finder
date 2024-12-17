package facet

import (
	"testing"

	"github.com/matst80/slask-finder/pkg/types"
)

func TestPresortedSorting(t *testing.T) {
	sortIndex := types.ByValue{
		types.Lookup{1, 6},
		types.Lookup{2, 5},
		types.Lookup{3, 4},
		types.Lookup{4, 3},
		types.Lookup{5, 2},
		types.Lookup{6, 1},
	}
	ids := types.ItemList{4: {}, 2: {}, 1: {}, 3: {}}

	expected := []uint{1, 2, 3, 4}
	i := 0
	for v := range sortIndex.SortMap(ids) {
		if v != expected[i] {
			t.Errorf("Expected %v, got %v", expected[i], v)
		}
		i++
	}

}
