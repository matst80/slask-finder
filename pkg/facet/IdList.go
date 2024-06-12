package facet

import (
	"log"
	"maps"
	"time"
)

type IdList map[uint]struct{}

func (r *IdList) SortedIds(srt SortIndex, maxItems int) []uint {
	return srt.SortMap(*r, maxItems)
}

func (a IdList) Intersect(b IdList) {
	start := time.Now()
	for id := range a {
		_, ok := b[id]
		if !ok {
			delete(a, id)
		}
	}
	log.Printf("Intersect took %v", time.Since(start))
}

func (i IdList) Merge(other *IdList) {
	maps.Copy(i, *other)
}

func MakeIntersectResult(r chan IdList, len int) IdList {

	if len == 0 {
		return IdList{}
	}
	first := <-r
	for i := 1; i < len; i++ {
		first.Intersect(<-r)
	}
	close(r)
	return first
}
