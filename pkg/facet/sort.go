package facet

import (
	"log"
	"time"
)

type SortIndex []int64

func (s *SortIndex) SortMap(ids IdList, breakAt int) []int64 {
	start := time.Now()

	l := min(len(ids), breakAt)
	sortedIds := make([]int64, l)
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
	log.Printf("Sorting took %v", time.Since(start))

	return sortedIds

}

type Lookup struct {
	Id    int64
	Value float64
}

type ByValue []Lookup

func (a ByValue) Len() int           { return len(a) }
func (a ByValue) Less(i, j int) bool { return a[i].Value < a[j].Value }
func (a ByValue) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
