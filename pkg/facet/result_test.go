package facet

import (
	"reflect"
	"testing"
)

func TestDuplicateIds(t *testing.T) {
	r := NewResult()
	r.Add(1, 2, 3, 4, 5, 6, 7)
	r.Add(4, 5, 6, 7, 8, 9, 10)
	ids := r.Ids()
	if !reflect.DeepEqual(ids, []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}) {
		t.Errorf("Expected [1, 2, 3, 4, 5, 6, 7, 8, 9, 10] but got %v", ids)
	}
}

func TestMergeResults(t *testing.T) {
	a := NewResult()
	b := NewResult()
	a.Add(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	b.Add(2, 3, 4, 5, 6, 7, 8, 9, 10, 11)
	a.Merge(b)
	ids := a.Ids()
	if !reflect.DeepEqual(ids, []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}) {
		t.Errorf("Expected [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11] but got %v", ids)
	}
}

func TestIntersectResults(t *testing.T) {
	a := NewResult()
	b := NewResult()
	a.Add(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	b.Add(2, 3, 4, 5, 6, 7, 8, 9, 10, 11)
	a.Intersect(b)
	ids := a.Ids()
	if !reflect.DeepEqual(ids, []int64{2, 3, 4, 5, 6, 7, 8, 9, 10}) {
		t.Errorf("Expected [2, 3, 4, 5, 6, 7, 8, 9, 10] but got %v", ids)
	}
}
