package sorting

import (
	"fmt"
	"slices"
	"strings"

	"github.com/matst80/slask-finder/pkg/types"
)

func ToMap(f *types.ByValue) map[uint32]float64 {
	m := make(map[uint32]float64)
	for _, item := range *f {
		m[uint32(item.Id)] = item.Value
	}
	return m
}

type StaticPositions map[int]uint

func (s *StaticPositions) ToString() string {
	ret := ""
	for key, value := range *s {
		ret += fmt.Sprintf("%d:%d,", key, value)
	}
	return ret
}

func (s *StaticPositions) FromString(data string) error {
	*s = make(map[int]uint)
	for item := range strings.SplitSeq(data, ",") {
		var key int
		var value uint
		_, err := fmt.Sscanf(item, "%d:%d", &key, &value)
		if err != nil {
			if err.Error() == "EOF" {
				return nil
			}
			return err
		}
		(*s)[key] = value
	}
	return nil
}

// LookupSortFunc returns a comparator suitable for slices.SortFunc over []types.Lookup.
// When ascending is false it sorts by Value descending.
// Ties on Value are broken deterministically by Id (ascending) to avoid jitter and
// remove the need for artificial epsilon adjustments.
func LookupSortFunc(ascending bool) func(a, b types.Lookup) int {
	if ascending {
		return func(a, b types.Lookup) int {
			if a.Value < b.Value {
				return -1
			}
			if a.Value > b.Value {
				return 1
			}
			// tie-break on Id
			if a.Id < b.Id {
				return -1
			}
			if a.Id > b.Id {
				return 1
			}
			return 0
		}
	}
	// descending
	return func(a, b types.Lookup) int {
		if a.Value > b.Value {
			return -1
		}
		if a.Value < b.Value {
			return 1
		}
		if a.Id < b.Id {
			return -1
		}
		if a.Id > b.Id {
			return 1
		}
		return 0
	}
}

// SortByValues keeps backward compatibility: sorts by Value descending.
func SortByValues(arr types.ByValue) {
	slices.SortFunc(arr, LookupSortFunc(false))
}

// SortByValuesOrder allows explicit ascending / descending control.
func SortByValuesOrder(arr types.ByValue, ascending bool) {
	slices.SortFunc(arr, LookupSortFunc(ascending))
}

/*
Benchmark note:

Add a new file sorting/sorter_benchmark_test.go with:

package sorting

import (
	"testing"
	"github.com/matst80/slask-finder/pkg/types"
)

func BenchmarkBaseSorterGetSort(b *testing.B) {
	s := NewBaseSorter("bench", func(it types.Item) float64 {
		return float64(it.GetPrice())
	}, false).(*BaseSorter)

	// mock items implementing types.Item would be added here (omitted for brevity)

	for n := 0; n < b.N; n++ {
		_ = s.GetSort()
	}
}

(You asked for a benchmark. It must live in a *_test.go file to run under `go test -bench=.`.)
*/
