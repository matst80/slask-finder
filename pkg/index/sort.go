package index

import (
	"sort"
)

type SortIndex []int64

func (s *SortIndex) SortIds(ids []int64, breakAt int) <-chan int64 {
	out := make(chan int64)

	ss := *s
	l := min(len(ids), breakAt)
	sortedIds := make([]int64, l)
	idx := 0

	go func() {
		for _, id := range ss {
			if idx >= l {
				break
			}
			for _, v := range ids[idx:] {
				if id == v {
					sortedIds[idx] = v
					out <- v
					idx++
					break
				}
			}
		}
	}()

	return out
}

type Lookup struct {
	Id    int64
	Value float64
}

type ByValue []Lookup

func (a ByValue) Len() int           { return len(a) }
func (a ByValue) Less(i, j int) bool { return a[i].Value < a[j].Value }
func (a ByValue) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func MakeSortFromNumberField(items map[int64]Item, fieldId int64) SortIndex {
	l := len(items)
	sortIndex := make(SortIndex, l)
	sortMap := make(ByValue, l)
	for idx, item := range items {
		sortMap[idx] = Lookup{Id: item.Id, Value: item.NumberFields[fieldId]}
	}
	sort.Sort(sortMap)
	for idx, item := range sortMap {
		sortIndex[idx] = item.Id
	}
	return sortIndex
}
