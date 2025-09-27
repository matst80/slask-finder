//go:build jsonv2

package types

import (
	json "encoding/json/v2"
	"testing"
)

func BenchmarkItemFieldsV2_Object_Marshal_Small(b *testing.B) {
	benchmarkV2Marshal(b, smallFields, false)
}
func BenchmarkItemFieldsV2_Object_Marshal_Med(b *testing.B) { benchmarkV2Marshal(b, medFields, false) }
func BenchmarkItemFieldsV2_Object_Marshal_Large(b *testing.B) {
	benchmarkV2Marshal(b, largeFields, false)
}
func BenchmarkItemFieldsV2_Object_Unmarshal_Small(b *testing.B) {
	benchmarkV2Unmarshal(b, smallFields, false)
}
func BenchmarkItemFieldsV2_Object_Unmarshal_Med(b *testing.B) {
	benchmarkV2Unmarshal(b, medFields, false)
}
func BenchmarkItemFieldsV2_Object_Unmarshal_Large(b *testing.B) {
	benchmarkV2Unmarshal(b, largeFields, false)
}

func BenchmarkItemFieldsV2_Compact_Marshal_Small(b *testing.B) {
	benchmarkV2Marshal(b, smallFields, true)
}
func BenchmarkItemFieldsV2_Compact_Marshal_Med(b *testing.B) { benchmarkV2Marshal(b, medFields, true) }
func BenchmarkItemFieldsV2_Compact_Marshal_Large(b *testing.B) {
	benchmarkV2Marshal(b, largeFields, true)
}
func BenchmarkItemFieldsV2_Compact_Unmarshal_Small(b *testing.B) {
	benchmarkV2Unmarshal(b, smallFields, true)
}
func BenchmarkItemFieldsV2_Compact_Unmarshal_Med(b *testing.B) {
	benchmarkV2Unmarshal(b, medFields, true)
}
func BenchmarkItemFieldsV2_Compact_Unmarshal_Large(b *testing.B) {
	benchmarkV2Unmarshal(b, largeFields, true)
}

func benchmarkV2Marshal(b *testing.B, f ItemFields, compact bool) {
	b.ReportAllocs()
	// toggle global flags
	EmitCompactItemFields = compact
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(f) // uses custom MarshalJSONTo via interface method
	}
}

func benchmarkV2Unmarshal(b *testing.B, f ItemFields, compact bool) {
	EmitCompactItemFields = compact
	data, _ := json.Marshal(f)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var dst ItemFields
		_ = json.Unmarshal(data, &dst)
	}
}
