package facet

import (
	"maps"
)

type ItemFields struct {
	Fields        map[uint]string  `json:"values"`
	DecimalFields map[uint]float64 `json:"numberValues"`
	IntegerFields map[uint]int     `json:"integerValues"`
}

type MatchList map[uint]*ItemFields

func (r *MatchList) SortedIds(srt *SortIndex, maxItems int) []uint {
	return srt.SortMatch(*r, maxItems)
}

func (a MatchList) Intersect(b MatchList) {
	for id := range a {
		_, ok := b[id]
		if !ok {
			delete(a, id)
		}
	}
}

func (i MatchList) Merge(other *MatchList) {
	maps.Copy(i, *other)
}

// func MakeIntersectResult(r chan MatchList, len int) *MatchList {

// 	if len == 0 {
// 		return &MatchList{}
// 	}
// 	first := <-r
// 	for i := 1; i < len; i++ {
// 		first.Intersect(<-r)
// 	}
// 	close(r)
// 	return &first
// }
