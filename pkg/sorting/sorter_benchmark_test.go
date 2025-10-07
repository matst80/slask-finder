package sorting

import (
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/matst80/slask-finder/pkg/types"
)

/*
Benchmark goals:

1. Measure the cost of GetSort (building + sorting the slice) for various collection sizes.
2. Cover both descending (default) and ascending (isReversed=true) variants.
3. Provide a realistic scoring function (price-based) while keeping mock item lightweight.

We separate item ingestion (ProcessItem) from the timed section; only GetSort
is inside the benchmark loop to focus on sorting/comparator overhead.

If you also want to benchmark incremental updates cost, add another benchmark
that mutates a subset of items each iteration before calling GetSort.
*/

// mockItem is a minimal implementation of types.Item sufficient for the sorter benchmarks.
type mockItem struct {
	id        types.ItemId
	price     int
	deleted   bool
	created   int64
	updated   int64
	sku       string
	title     string
	numberMap map[types.FacetId]float64
	stringMap map[types.FacetId]string
}

func newMockItem(id int, price int) *mockItem {
	now := time.Now().Unix()
	return &mockItem{
		id:        types.ItemId(id),
		price:     price,
		created:   now,
		updated:   now,
		sku:       fmt.Sprintf("SKU-%d", id),
		title:     fmt.Sprintf("Item %d", id),
		numberMap: nil,
		stringMap: nil,
	}
}

func (m *mockItem) GetId() types.ItemId                        { return m.id }
func (m *mockItem) GetSku() string                             { return m.sku }
func (m *mockItem) GetStock() map[string]uint32                { return nil }
func (m *mockItem) HasStock() bool                             { return true }
func (m *mockItem) IsDeleted() bool                            { return m.deleted }
func (m *mockItem) IsSoftDeleted() bool                        { return false }
func (m *mockItem) GetPropertyValue(name string) any           { return nil }
func (m *mockItem) GetPrice() int                              { return m.price }
func (m *mockItem) GetDiscount() int                           { return 0 }
func (m *mockItem) GetRating() (int, int)                      { return 0, 0 }
func (m *mockItem) GetStringFields() map[types.FacetId]string  { return m.stringMap }
func (m *mockItem) GetNumberFields() map[types.FacetId]float64 { return m.numberMap }
func (m *mockItem) GetStringFieldValue(id types.FacetId) (string, bool) {
	v, ok := m.stringMap[id]
	return v, ok
}
func (m *mockItem) GetStringsFieldValue(id types.FacetId) ([]string, bool) { return nil, false }
func (m *mockItem) GetNumberFieldValue(id types.FacetId) (float64, bool) {
	v, ok := m.numberMap[id]
	return v, ok
}
func (m *mockItem) GetLastUpdated() int64              { return m.updated }
func (m *mockItem) GetCreated() int64                  { return m.created }
func (m *mockItem) GetTitle() string                   { return m.title }
func (m *mockItem) ToString() string                   { return m.title }
func (m *mockItem) ToStringList() []string             { return []string{m.title} }
func (m *mockItem) CanHaveEmbeddings() bool            { return false }
func (m *mockItem) GetEmbeddingsText() (string, error) { return "", nil }
func (m *mockItem) Write(w io.Writer) (int, error)     { return w.Write([]byte(m.title)) }

// prepareSorter builds and primes a sorter with N mock items.
// The scoring function just returns float64(price).
func prepareSorter(n int, reversed bool) Sorter {
	sorter := NewBaseSorter("price", func(it types.Item) float64 {
		return float64(it.GetPrice())
	}, reversed)

	// Ingest items (simulate a range of price patterns to cause ties)
	for i := 0; i < n; i++ {
		// Price pattern ensures repetitions (ties) to exercise tie-break logic:
		price := (i * 37) % 1000 // pseudo-random-ish deterministic spread in [0,999]
		sorter.ProcessItem(newMockItem(i, price))
	}
	return sorter
}

var benchmarkSizes = []int{
	100,
	1_000,
	10_000,
	50_000,
	100_000,
}

// BenchmarkBaseSorterGetSortDescending benchmarks GetSort with descending order (default).
func BenchmarkBaseSorterGetSortDescending(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("N=%d", size), func(b *testing.B) {
			sorter := prepareSorter(size, false)
			// Ensure initial state considered clean
			_ = sorter.GetSort()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = sorter.GetSort()
			}
		})
	}
}

// BenchmarkBaseSorterGetSortAscending benchmarks GetSort with ascending order (isReversed=true).
func BenchmarkBaseSorterGetSortAscending(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("N=%d", size), func(b *testing.B) {
			sorter := prepareSorter(size, true)
			_ = sorter.GetSort()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = sorter.GetSort()
			}
		})
	}
}

// BenchmarkBaseSorterGetSortWithDirty simulates mutations each iteration to include scoring cost.
func BenchmarkBaseSorterGetSortWithDirty(b *testing.B) {
	const mutateEvery = 10 // mutate a handful of items before each sort
	for _, size := range []int{10_000, 50_000} {
		b.Run(fmt.Sprintf("N=%d", size), func(b *testing.B) {
			sorter := prepareSorter(size, false)

			// Pre-fetch slice of items to mutate (just reusing ids)
			items := make([]*mockItem, mutateEvery)
			for i := 0; i < mutateEvery; i++ {
				items[i] = newMockItem(i, i%1000)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Simulate price changes (mutation) before sort
				for j := 0; j < mutateEvery; j++ {
					it := items[j]
					// Change price deterministically
					it.price = (it.price + 17) % 1000
					sorter.ProcessItem(it)
				}
				_ = sorter.GetSort()
			}
		})
	}
}
