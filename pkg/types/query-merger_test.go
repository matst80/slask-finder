package types

import (
	"context"
	"testing"
)

func TestQueuedMerger(t *testing.T) {
	// Create a new QueryMerger instance (roaring-backed ItemList)
	var result ItemList
	merger := NewQueryMerger(context.Background(), &result)

	// First constraint: {1,2}
	merger.Add(func(_ context.Context) *ItemList {
		l := &ItemList{}
		l.AddId(1)
		l.AddId(2)
		return l
	})

	// Second constraint: {2,3} -> intersection should be {2}
	merger.Add(func(_ context.Context) *ItemList {
		l := &ItemList{}
		l.AddId(2)
		l.AddId(3)
		return l
	})

	merger.Wait()

	if result.Len() != 1 || !result.Contains(2) {
		t.Errorf("expected intersection to be {2}, got len=%d contains2=%v", result.Len(), result.Contains(2))
	}
}
