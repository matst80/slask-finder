package index

import (
	"reflect"
	"testing"

	"tornberg.me/facet-search/pkg/facet"
)

func TestIndexMatch(t *testing.T) {
	i := NewIndex()
	item := Item{
		Id: 1,
		Fields: []facet.StringFieldReference{
			{Value: "test", Id: 1},
			{Value: "hej", Id: 2},
		},
		NumberFields: []facet.NumberFieldReference{
			{Value: 1, Id: 3},
		},
	}
	i.AddItem(item)
	matching := i.Match([]StringSearch{{Id: 1, Value: "test"}}, []NumberSearch{{Min: 1, Max: 2}})
	if !reflect.DeepEqual(matching, []int64{1}) {
		t.Errorf("Expected [1] but got %v", matching)
	}
}

func CreateIndex() *Index {
	i := NewIndex()
	i.AddField(1, facet.Field{Name: "first", Description: "first field"})
	i.AddField(2, facet.Field{Name: "other", Description: "other field"})
	i.AddNumberField(3, facet.Field{Name: "number", Description: "number field"})

	i.AddItem(Item{
		Id:    1,
		Title: "item1",
		Fields: []facet.StringFieldReference{
			{Value: "test", Id: 1},
			{Value: "hej", Id: 2},
		},
		NumberFields: []facet.NumberFieldReference{
			{Value: 1, Id: 3},
		},
		Props: map[string]string{
			"test": "test",
		},
	})
	i.AddItem(Item{
		Id:    2,
		Title: "item2",
		Fields: []facet.StringFieldReference{
			{Value: "test", Id: 1},
			{Value: "slask", Id: 2},
		},
		NumberFields: []facet.NumberFieldReference{
			{Value: 1, Id: 3},
		},
		Props: map[string]string{
			"hej": "hej",
		},
	})
	return i
}

func TestHasItem(t *testing.T) {
	i := CreateIndex()

	if !i.HasItem(1) {
		t.Errorf("Expected to have item with id 1")
	}
	if i.HasItem(3) {
		t.Errorf("Expected to not have item with id 2")
	}
}

func TestHasFields(t *testing.T) {
	i := CreateIndex()
	if len(i.Fields) != 2 {
		t.Errorf("Expected to have 2 fields")
	}
	if len(i.NumberFields) != 1 {
		t.Errorf("Expected to 1 number field")
	}
	field, ok := i.Fields[int64(1)]
	if !ok {
		t.Errorf("Expected to have field with id 1, got %v", i.Fields)
	}
	if len(field.Values()) != 2 {
		t.Errorf("Expected to have 2 values in field 1, got %v", field.Values())
	}
}

func TestMultipleIndexMatch(t *testing.T) {
	i := CreateIndex()
	matching := i.Match([]StringSearch{{Id: 1, Value: "test"}}, []NumberSearch{{Id: 3, Min: 1, Max: 2}})
	if !reflect.DeepEqual(matching, []int64{1, 2}) {
		t.Errorf("Expected [1,2] but got %v", matching)
	}
}

func TestGetMatchItems(t *testing.T) {
	i := CreateIndex()
	matching := i.Match([]StringSearch{{Id: 1, Value: "test"}}, []NumberSearch{{Id: 3, Min: 1, Max: 2}})
	items := i.GetItems(matching)
	if items[0].Id != 1 || items[1].Id != 2 {
		t.Errorf("Expected ids [1,2] but got %v", items)
	}
	if items[0].Title != "item1" || items[1].Title != "item2" {
		t.Errorf("Expected titles [item1, item2] but got %v", items)
	}
	// if items[0].Fields[0].Value != "test" || items[1].Fields[0].Value != "hoj" {
	// 	t.Errorf("Expected fields [test, hoj] but got %v", items)
	// }
}

func TestGetFacetsFromResultIds(t *testing.T) {
	i := CreateIndex()
	matching := i.Match([]StringSearch{{Id: 1, Value: "test"}}, []NumberSearch{{Min: 1, Max: 2}})
	facets := i.GetFacetsFromResultIds(matching)
	if len(facets.Fields) != 2 {
		t.Errorf("Expected 2 fields but got %v", facets.Fields)
	}
	if len(facets.NumberFields) != 1 {
		t.Errorf("Expected 1 number fields but got %v", facets.NumberFields)
	}
	// if len(facets.Fields[0].Values()) != 4 {
	// 	t.Errorf("Expected 4 value in field 1 but got %v", facets.Fields[1].Values())
	// }
	// if len(facets.NumberFields[0].Values()) != 1 {
	// 	t.Errorf("Expected 1 value in field 1 but got %v", facets.NumberFields[1].Values())
	// }
}
