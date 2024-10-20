package index

import (
	"fmt"
	"sort"
	"strings"

	"tornberg.me/facet-search/pkg/types"
)

func ToMap(f *types.ByValue) map[uint]float64 {
	m := make(map[uint]float64)
	for _, item := range *f {
		m[item.Id] = item.Value
	}
	return m
}

func ToSortIndex(f *types.ByValue, reversed bool) *types.SortIndex {
	l := len(*f)
	if reversed {
		sort.Sort(sort.Reverse(*f))
	} else {
		sort.Sort(*f)
	}

	sortIndex := make(types.SortIndex, l)
	for idx, item := range *f {
		sortIndex[idx] = item.Id
	}
	return &sortIndex
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
	for _, item := range strings.Split(data, ",") {
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

type SortOverride map[uint]float64

func (s *SortOverride) ToString() string {
	ret := ""
	for key, value := range *s {
		ret += fmt.Sprintf("%d:%f,", key, value)
	}
	return ret
}

func (s *SortOverride) FromString(data string) error {
	*s = make(map[uint]float64)
	for _, item := range strings.Split(data, ",") {
		var key uint
		var value float64
		_, err := fmt.Sscanf(item, "%d:%f", &key, &value)
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
