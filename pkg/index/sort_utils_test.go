package index

import "testing"

func TestSortOverride(t *testing.T) {
	s := SortOverride{
		1: 0.5,
		2: 0.3,
		3: 0.7,
		4: 100.0,
	}
	b := s.ToSortedLookup()
	if len(b) != 4 {
		t.Errorf("Expected 4 items, got %d", len(b))
	}
	if b[0].Id != 4 {
		t.Errorf("Expected 4 as first item, got %d", b[0].Id)
	}
	if b[1].Id != 3 {
		t.Errorf("Expected 3 as second item, got %d", b[1].Id)
	}
	if b[2].Id != 1 {
		t.Errorf("Expected 1 as third item, got %d", b[2].Id)
	}
	if b[3].Id != 2 {
		t.Errorf("Expected 2 as fourth item, got %d", b[3].Id)
	}
	if b[0].Value != 100.0 {
		t.Errorf("Expected 100.0 as first value, got %f", b[0].Value)
	}

}
