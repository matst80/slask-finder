package types

import (
	stdjson "encoding/json"
	"math/rand"
	"testing"
	"time"
)

// makeTestItemFields creates a representative ItemFields map with a mix of value kinds.
func makeTestItemFields(n int) ItemFields {
	rand.Seed(42)
	f := make(ItemFields, n)
	for i := 0; i < n; i++ {
		id := uint(i + 1)
		switch i % 6 {
		case 0:
			f[id] = i * 10
		case 1:
			f[id] = float64(i) * 1.25
		case 2:
			f[id] = (i%2 == 0)
		case 3:
			f[id] = time.Unix(int64(1700000000+i), 0).UTC().Format(time.RFC3339)
		case 4:
			f[id] = []string{"a", "b", "c"}
		case 5:
			f[id] = map[string]any{"k": i}
		}
	}
	return f
}

// Global test data reused across benchmarks to avoid regeneration cost.
var (
	smallFields = makeTestItemFields(16)
	medFields   = makeTestItemFields(64)
	largeFields = makeTestItemFields(256)
)

func BenchmarkItemFieldsStd_Marshal_Small(b *testing.B)   { benchmarkStdMarshal(b, smallFields) }
func BenchmarkItemFieldsStd_Marshal_Med(b *testing.B)     { benchmarkStdMarshal(b, medFields) }
func BenchmarkItemFieldsStd_Marshal_Large(b *testing.B)   { benchmarkStdMarshal(b, largeFields) }
func BenchmarkItemFieldsStd_Unmarshal_Small(b *testing.B) { benchmarkStdUnmarshal(b, smallFields) }
func BenchmarkItemFieldsStd_Unmarshal_Med(b *testing.B)   { benchmarkStdUnmarshal(b, medFields) }
func BenchmarkItemFieldsStd_Unmarshal_Large(b *testing.B) { benchmarkStdUnmarshal(b, largeFields) }

func benchmarkStdMarshal(b *testing.B, f ItemFields) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = stdjson.Marshal(f)
	}
}

func benchmarkStdUnmarshal(b *testing.B, f ItemFields) {
	data, _ := stdjson.Marshal(f)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var dst ItemFields
		_ = stdjson.Unmarshal(data, &dst)
	}
}
