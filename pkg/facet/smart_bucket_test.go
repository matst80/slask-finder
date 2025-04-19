package facet

import (
	"testing"
)

func TestSmartBucket(t *testing.T) {
	bucket := NewSmartBucket(5)
	bucket.AddValueLink(1, 1)
	bucket.AddValueLink(2, 2)
	bucket.AddValueLink(3, 3)
	bucket.AddValueLink(4, 4)
	bucket.AddValueLink(5, 5)
	bucket.AddValueLink(6, 6)
	bucket.AddValueLink(7, 7)
	bucket.AddValueLink(70, 70)
	bucket.AddValueLink(40, 40)

	if bucket.MinValue != 1 {
		t.Errorf("Expected MinValue to be 1, got %d", bucket.MinValue)
	}
	if bucket.MaxValue != 6 {
		t.Errorf("Expected MaxValue to be 6, got %d", bucket.MaxValue)
	}
	if bucket.Count != 6 {
		t.Errorf("Expected Count to be 6, got %d", bucket.Count)
	}
	ids := bucket.Match(3, 40)
	if len(ids) != 6 {
		t.Errorf("Expected 6 ids, got %d", len(ids))
	}
}
