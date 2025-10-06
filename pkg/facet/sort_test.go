package facet

import (
	"testing"

	"github.com/matst80/slask-finder/pkg/types"
)

func TestPresortedSorting(t *testing.T) {
	sortIndex := types.ByValue{
		types.Lookup{Id: 1, Value: 6},
		types.Lookup{Id: 2, Value: 5},
		types.Lookup{Id: 3, Value: 4},
		types.Lookup{Id: 4, Value: 3},
		types.Lookup{Id: 5, Value: 2},
		types.Lookup{Id: 6, Value: 1},
	}
	ids := types.NewItemList()
	ids.AddId(4)
	ids.AddId(2)
	ids.AddId(1)
	ids.AddId(3)

	expected := []uint32{1, 2, 3, 4}
	i := 0
	for v := range sortIndex.SortBitmap(*ids.Bitmap()) {
		if v != expected[i] {
			t.Errorf("Expected %v, got %v", expected[i], v)
		}
		i++
	}

}
