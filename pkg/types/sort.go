package types

import (
	"cmp"
	"fmt"
	"iter"
	"log"
	"strconv"
	"strings"
)

type SortIndex []uint

func (s *SortIndex) GetScore(id uint) float64 {
	for i, v := range *s {
		if v == id {
			return float64(i)
		}
	}
	return -1
}

func (s *SortIndex) Add(id uint) {
	*s = append(*s, id)
}

func (s *SortIndex) Remove(id uint) {
	for i, v := range *s {
		if v == id {
			*s = append((*s)[:i], (*s)[i+1:]...)
			return
		}
	}
}

func (s *SortIndex) FromString(data string) error {
	for str := range strings.SplitSeq(data, ",") {
		i, err := strconv.ParseUint(str, 10, 64)
		if err != nil {
			return err
		}
		s.Add(uint(i))
	}
	return nil
}

func (s *SortIndex) ToString() string {
	var buffer strings.Builder
	for i, id := range *s {
		buffer.WriteString(fmt.Sprintf("%d", id))
		if i != len(*s)-1 {
			buffer.WriteString(",")
		}
	}
	return buffer.String()
}

func (s *SortIndex) SortMapWithStaticPositions(ids ItemList, staticPositions map[int]uint) iter.Seq[uint] {

	if s == nil {
		log.Printf("SortIndex is nil")
		return func(yield func(uint) bool) {
			for id := range ids {
				if !yield(id) {
					break
				}

			}
		}
	}

	return func(yield func(uint) bool) {
		idx := 0
		for _, id := range *s {

			if sp, ok := staticPositions[idx]; ok {
				_, ok := ids[sp]
				if ok {
					if !yield(sp) {
						break
					}
					idx++
				}
			}

			_, ok := ids[id]
			if ok {
				if !yield(id) {
					break
				}
				idx++
			}
		}
	}
	//return sortedIds
}

func (s *ByValue) SortMap(ids ItemList) iter.Seq[uint] {

	if s == nil {
		log.Printf("SortIndex is nil")
		return func(yield func(uint) bool) {
			for id := range ids {
				if !yield(id) {
					break
				}
			}
		}
	}

	return func(yield func(uint) bool) {
		for _, v := range *s {
			_, ok := ids[v.Id]
			if ok {
				if !yield(v.Id) {
					break
				}
			}
		}
	}
}

type Lookup struct {
	Id    uint
	Value float64
}

type ByValue []Lookup

func (a ByValue) Len() int           { return len(a) }
func (a ByValue) Less(i, j int) bool { return a[i].Value < a[j].Value }
func (a ByValue) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func LookUpReversed(a, b Lookup) int {
	return cmp.Compare(b.Value, a.Value)
}

func LookUpNormal(a, b Lookup) int {
	return cmp.Compare(a.Value, b.Value)
}
