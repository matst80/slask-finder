package facet

import (
	"log"
)

type SortIndex []uint

func (s *SortIndex) SortMap(ids IdList, breakAt int) []uint {

	if s == nil {
		log.Printf("SortIndex is nil")
		return []uint{}
	}

	l := min(len(ids), breakAt)
	sortedIds := make([]uint, l)
	idx := 0

	for _, id := range *s {
		if idx >= l {
			break
		}
		_, ok := ids[id]
		if ok {
			sortedIds[idx] = id

			idx++

		}
	}

	return sortedIds

}

func (s *SortIndex) SortMatch(ids MatchList, breakAt int) []uint {

	if s == nil {
		log.Printf("SortIndex is nil")
		return []uint{}
	}

	l := min(len(ids), breakAt)
	sortedIds := make([]uint, l)
	idx := 0

	for _, id := range *s {
		if idx >= l {
			break
		}
		_, ok := ids[id]
		if ok {
			sortedIds[idx] = id

			idx++

		}
	}

	return sortedIds

}

type Lookup struct {
	Id    uint
	Value float64
}

type ByValue []Lookup

func (a ByValue) Len() int           { return len(a) }
func (a ByValue) Less(i, j int) bool { return a[i].Value < a[j].Value }
func (a ByValue) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
