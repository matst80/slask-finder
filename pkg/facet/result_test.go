package facet

import (
	"reflect"
	"testing"
)

func TestDuplicateIds(t *testing.T) {
	r := Result{}
	r.Add(1, 2, 3, 4, 5, 6, 7)
	r.Add(4, 5, 6, 7, 8, 9, 10)
	if !reflect.DeepEqual(r.Ids, []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}) {
		t.Errorf("Expected [1, 2, 3, 4, 5, 6, 7, 8, 9, 10] but got %v", r.Ids)
	}
}

func TestMergeResults(t *testing.T) {
	a := Result{}
	b := Result{}
	a.Add(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	b.Add(2, 3, 4, 5, 6, 7, 8, 9, 10, 11)
	a.Merge(b)
	if !reflect.DeepEqual(a.Ids, []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}) {
		t.Errorf("Expected [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11] but got %v", a.Ids)
	}
}

func TestIntersectResults(t *testing.T) {
	a := Result{}
	b := Result{}
	a.Add(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	b.Add(2, 3, 4, 5, 6, 7, 8, 9, 10, 11)
	a.Intersect(b)
	if !reflect.DeepEqual(a.Ids, []int64{2, 3, 4, 5, 6, 7, 8, 9, 10}) {
		t.Errorf("Expected [2, 3, 4, 5, 6, 7, 8, 9, 10] but got %v", a.Ids)
	}
}
