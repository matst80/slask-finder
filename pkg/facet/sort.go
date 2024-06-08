package facet

import (
	"log"
	"time"
)

type SortIndex []int64

func (s *SortIndex) SortIds(ids []int64, breakAt int) []int64 {
	start := time.Now()
	ss := make([]int64, len(*s))
	copy(ss, *s)
	l := min(len(ids), breakAt)
	sortedIds := make([]int64, l)
	idx := 0

	for _, id := range ss {
		if idx >= l {
			break
		}
		for _, v := range ids[idx:] {
			if id == v {
				sortedIds[idx] = v

				idx++
				break
			}
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
