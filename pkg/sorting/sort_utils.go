package sorting

import (
	"cmp"
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

func SortByValues(arr types.ByValue) {
	slices.SortFunc(arr, func(a, b types.Lookup) int {
		return cmp.Compare(b.Value, a.Value)
	})
}
