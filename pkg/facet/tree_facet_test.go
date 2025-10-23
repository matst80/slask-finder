package facet

import (
	"testing"

	"github.com/matst80/slask-finder/pkg/types"
)

func TestTreeFacetAddValueLink(t *testing.T) {
	field := EmptyTreeValueField(&types.BaseField{Id: 1, Name: "tree", Searchable: true})
	tf := field.(*TreeField)

	// Add one path
	field.AddValueLink([]string{"a", "b", "c"}, 1)

	// Check tree structure
	if len(tf.Children) != 1 {
		t.Errorf("Expected 1 top-level child, got %d", len(tf.Children))
	}

	a, ok := tf.Children["a"]
	if !ok {
		t.Fatal("Expected 'a' child")
	}
	if !a.ids.Contains(uint32(1)) {
		t.Errorf("'a' ids: expected 1, got %v", a.ids.ToSlice())
	}

	b, ok := a.Children["b"]
	if !ok {
		t.Fatal("Expected 'b' child under 'a'")
	}
	if !b.ids.Contains(uint32(1)) {
		t.Errorf("'b' ids: expected 1, got %v", b.ids.ToSlice())
	}

	c, ok := b.Children["c"]
	if !ok {
		t.Fatal("Expected 'c' child under 'b'")
	}
	if !c.ids.Contains(uint32(1)) {
		t.Errorf("'c' ids: expected 1, got %v", c.ids.ToSlice())
	}
}

func TestTreeFacetMatch(t *testing.T) {
	field := EmptyTreeValueField(&types.BaseField{Id: 1, Name: "tree", Searchable: true})

	// Add some paths
	field.AddValueLink([]string{"a", "b", "c"}, 1)
	field.AddValueLink([]string{"a", "b", "d"}, 2)
	field.AddValueLink([]string{"a", "e"}, 3)
	field.AddValueLink([]string{"x", "y"}, 4)

	// Test matches
	tests := []struct {
		path     []string
		expected []types.ItemId
	}{
		{[]string{"a"}, []types.ItemId{1, 2, 3}},
		{[]string{"a", "b"}, []types.ItemId{1, 2}},
		{[]string{"a", "b", "c"}, []types.ItemId{1}},
		{[]string{"a", "b", "d"}, []types.ItemId{2}},
		{[]string{"a", "e"}, []types.ItemId{3}},
		{[]string{"x"}, []types.ItemId{4}},
		{[]string{"x", "y"}, []types.ItemId{4}},
		{[]string{"z"}, []types.ItemId{}}, // non-existing
		{[]string{}, []types.ItemId{}},    // empty
	}

	for _, test := range tests {
		result := field.Match(test.path)
		if !containsAll(result, test.expected) || result.Len() != len(test.expected) {
			t.Errorf("Match %v: expected %v, got %v", test.path, test.expected, result.ToSlice())
		}
	}
}

func TestTreeFacetMatchInterface(t *testing.T) {
	field := EmptyTreeValueField(&types.BaseField{Id: 1, Name: "tree", Searchable: true})

	field.AddValueLink([]interface{}{"a", "b"}, 1)

	result := field.Match([]interface{}{"a", "b"})
	if !result.Contains(uint32(1)) {
		t.Errorf("Expected id 1, got %v", result.ToSlice())
	}
}

func containsAll(list *types.ItemList, ids []types.ItemId) bool {
	for _, id := range ids {
		if !list.Contains(uint32(id)) {
			return false
		}
	}
	return true
}
