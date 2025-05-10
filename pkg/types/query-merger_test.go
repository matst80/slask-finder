package types

import (
	"maps"
	"testing"
)

func TestQueuedMerger(t *testing.T) {
	// Create a new QueryMerger instance
	result := make(ItemList)
	merger := NewQueryMerger(&result)

	// Add a function to the merger that returns an ItemList
	merger.Add(func() *ItemList {
		return &ItemList{1: {}, 2: {}}
	})

	// Add another function to the merger that returns an ItemList
	merger.Add(func() *ItemList {
		return &ItemList{2: {}, 3: {}}
	})

	// Wait for all goroutines to finish
	merger.Wait()

	// Check the result
	expected := ItemList{2: {}}
	if !maps.Equal(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}
