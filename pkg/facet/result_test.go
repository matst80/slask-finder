package facet

import (
	"testing"
)

func makeIdList(ids ...int64) IdList {
	idList := make(IdList)
	for _, id := range ids {
		idList[id] = struct{}{}
	}
	return idList
}

func matchAll(list IdList, ids ...int64) bool {
	for _, id := range ids {
		if _, ok := list[id]; !ok {
			return false
		}
	}
	return true
}

func TestDuplicateIds(t *testing.T) {
	r := NewResult()
	r.Add(makeIdList(1, 2, 3, 4, 5, 6, 7))
	r.Add(makeIdList(4, 5, 6, 7, 8, 9, 10))
	ids := r.Ids()
	if !matchAll(r.ids, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10) {
		t.Errorf("Expected [1, 2, 3, 4, 5, 6, 7, 8, 9, 10] but got %v", ids)
	}
}

func TestMergeResults(t *testing.T) {
	a := NewResult()
	b := NewResult()
	a.Add(makeIdList(1, 2, 3, 4, 5, 6, 7, 8, 9, 10))
	b.Add(makeIdList(2, 3, 4, 5, 6, 7, 8, 9, 10, 11))
	a.Merge(b)

	if !matchAll(a.ids, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11) {
		t.Errorf("Expected [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11] but got %v", a.Ids())
	}
}

func TestIntersectResults(t *testing.T) {
	a := NewResult()
	b := NewResult()
	a.Add(makeIdList(1, 2, 3, 4, 5, 6, 7, 8, 9, 10))
	b.Add(makeIdList(2, 3, 4, 5, 6, 7, 8, 9, 10, 11))
	a.Intersect(b)

	if !matchAll(a.ids, 2, 3, 4, 5, 6, 7, 8, 9, 10) {
		t.Errorf("Expected [2, 3, 4, 5, 6, 7, 8, 9, 10] but got %v", a.Ids())
	}
}
