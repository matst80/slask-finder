package types

import (
	"fmt"
	"slices"
	"strings"
)

type SortOverride map[uint]float64

func (s *SortOverride) ToString() string {
	ret := ""
	for key, value := range *s {
		ret += fmt.Sprintf("%d:%f,", key, value)
	}
	return ret
}

func (s *SortOverride) Set(id uint, value float64) {
	(*s)[id] = value
}

func (s *SortOverride) FromString(data string) error {
	var key uint
	var value float64
	for item := range strings.SplitSeq(data, ",") {
		_, err := fmt.Sscanf(item, "%d:%f", &key, &value)
		if err != nil {
			if err.Error() == "EOF" {
				return nil
			}
			return err
		}
		s.Set(key, value)
	}
	return nil
}

func (s *SortOverride) ToSortedLookup() ByValue {

	return slices.SortedFunc(func(yield func(lookup Lookup) bool) {
		for id, value := range *s {
			if !yield(Lookup{Id: id, Value: value}) {
				break
			}
		}
	}, LookUpReversed)

}
