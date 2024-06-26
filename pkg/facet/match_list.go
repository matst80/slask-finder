package facet

import (
	"maps"
)

type KeyFieldValue struct {
	Value string `json:"value"`
	Id    uint   `json:"id"`
}

type DecimalFieldValue struct {
	Value float64 `json:"value"`
	Id    uint    `json:"id"`
}

type IntegerFieldValue struct {
	Value int  `json:"value"`
	Id    uint `json:"id"`
}

type ItemFields struct {
	Fields        []KeyFieldValue     `json:"values"`
	DecimalFields []DecimalFieldValue `json:"numberValues"`
	IntegerFields []IntegerFieldValue `json:"integerValues"`
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
