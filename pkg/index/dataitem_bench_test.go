package index

import (
	"encoding/json"
	"testing"

	"github.com/matst80/slask-finder/pkg/types"
)

// sampleJSON should reflect a realistic DataItem payload (object form for values)
var sampleJSON = []byte(`{
  "id":12345,
  "sku":"ABC-123",
  "title":"Example Product Title",
  "stock":{"online":"5","store":"2"},
  "values":{
    "3":5,
    "4":9999,
    "5":12999,
    "6":4,
    "7":25,
    "8":"Some bullet points here",
    "9":"Elgiganten",
    "10":"Normal",
    "11":["tag1","tag2"],
    "12":1.23,
    "13":true,
    "14":{"nested":"obj","arr":[1,2,3]}
  }
}`)

// construct a DataItem instance equivalent to sampleJSON for marshal benchmarks
func makeSampleDataItem() *DataItem {
	return &DataItem{BaseItem: &BaseItem{Id: 12345, Sku: "ABC-123", Title: "Example Product Title", Stock: map[string]string{"online": "5", "store": "2"}},
		Fields: types.ItemFields{
			3:  5,
			4:  9999,
			5:  12999,
			6:  4,
			7:  25,
			8:  "Some bullet points here",
			9:  "Elgiganten",
			10: "Normal",
			11: []string{"tag1", "tag2"},
			12: 1.23,
			13: true,
			14: map[string]interface{}{"nested": "obj", "arr": []int{1, 2, 3}},
		},
	}
}

func BenchmarkDataItemStd_Unmarshal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var di DataItem
		if err := json.Unmarshal(sampleJSON, &di); err != nil {
			b.Fatalf("unmarshal failed: %v", err)
		}
	}
}

func BenchmarkDataItemStd_Marshal(b *testing.B) {
	di := makeSampleDataItem()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := json.Marshal(di)
		if err != nil {
			b.Fatalf("marshal failed: %v", err)
		}
		if i == 0 {
			b.SetBytes(int64(len(out)))
			b.Logf("std json size=%d bytes", len(out))
		}
	}
}
