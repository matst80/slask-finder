//go:build jsonv2

package index

import (
	json "encoding/json/v2"
	"testing"

	"github.com/matst80/slask-finder/pkg/types"
)

func BenchmarkDataItemV2_Object_Unmarshal(b *testing.B) {
	types.EmitCompactItemFields = false
	for i := 0; i < b.N; i++ {
		var di DataItem
		if err := json.Unmarshal(sampleJSON, &di); err != nil {
			b.Fatalf("unmarshal failed: %v", err)
		}
	}
}

func BenchmarkDataItemV2_Object_Marshal(b *testing.B) {
	types.EmitCompactItemFields = false
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
			b.Logf("v2 object size=%d bytes", len(out))
		}
	}
}

func BenchmarkDataItemV2_Compact_Marshal(b *testing.B) {
	types.EmitCompactItemFields = true
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
			b.Logf("v2 compact size=%d bytes", len(out))
		}
	}
}

func BenchmarkDataItemV2_Compact_Unmarshal(b *testing.B) {
	types.EmitCompactItemFields = true
	di := makeSampleDataItem()
	compactJSON, err := json.Marshal(di)
	if err != nil {
		b.Fatalf("pre-marshal: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out DataItem
		if err := json.Unmarshal(compactJSON, &out); err != nil {
			b.Fatalf("unmarshal failed: %v", err)
		}
	}
}
