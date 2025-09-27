//go:build jsonv2

package index

import (
	json "encoding/json/v2"
	"testing"

	"github.com/matst80/slask-finder/pkg/types"
)

func makeAllocTestDataItem() *DataItem {
	return &DataItem{BaseItem: &BaseItem{Id: 42, Sku: "SKU-42", Title: "Alloc Test", Stock: map[string]string{"online": "7", "store": "3"}},
		Fields: types.ItemFields{3: 5, 4: 9999, 5: 12999, 6: 4, 7: 25, 8: "Some bp", 9: "Elgiganten", 10: "Normal", 11: []interface{}{"a", "b", "c"}, 12: 1.23, 13: true},
	}
}

func BenchmarkDataItemV2_Custom_Marshal(b *testing.B) {
	types.EmitCompactItemFields = true
	di := makeAllocTestDataItem()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := json.Marshal(di); err != nil {
			b.Fatalf("marshal failed: %v", err)
		}
	}
}
