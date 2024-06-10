package index

import (
	"reflect"
	"testing"

	"tornberg.me/facet-search/pkg/facet"
)

func matchAll(list facet.IdList, ids ...int64) bool {
	for _, id := range ids {
		if _, ok := list[id]; !ok {
			return false
		}
	}
	return true
}

func TestIndexMatch(t *testing.T) {
	i := NewIndex()
	i.AddKeyField(&facet.Field[string]{Id: 1, Name: "first", Description: "first field"})
	i.AddKeyField(&facet.Field[string]{Id: 2, Name: "other", Description: "other field"})
	i.AddDecimalField(&facet.NumberField[float64]{Id: 3, Name: "number", Description: "number field"})
	item := Item{
		Id: 1,
		Fields: map[int64]string{
			1: "test",
			2: "hej",
		},
		DecimalFields: map[int64]float64{
			3: 1,
		},
	}
	i.AddItem(item)
	query := Filters{
		StringFilter: []StringSearch{{Id: 1, Value: "test"}},
		NumberFilter: []NumberSearch{{Id: 3, Min: 1, Max: 2}},
		BoolFilter:   []BoolSearch{},
	}
	matching := i.Match(&query)
	if !matchAll(*matching.GetMap(), 1) {
		t.Errorf("Expected [1] but got %v", matching)
	}
}

func CreateIndex() *Index {
	i := NewIndex()
	i.AddKeyField(&facet.Field[string]{Id: 1, Name: "first", Description: "first field"})
	i.AddKeyField(&facet.Field[string]{Id: 2, Name: "other", Description: "other field"})
	i.AddDecimalField(&facet.NumberField[float64]{Id: 3, Name: "number", Description: "number field"})

	i.AddItem(Item{
		Id:    1,
		Title: "item1",
		Fields: map[int64]string{
			1: "test",
			2: "hej",
		},
		DecimalFields: map[int64]float64{
			3: 1,
		},
		Props: map[string]ItemProp{
			"test":  "test",
			"slask": 1,
		},
	})
	i.AddItem(Item{
		Id:    2,
		Title: "item2",
		Fields: map[int64]string{
			1: "test",
			2: "slask",
		},
		DecimalFields: map[int64]float64{
			3: 1,
		},
		Props: map[string]ItemProp{
			"hej": "hej",
			"ja":  true,
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
	if len(i.KeyFacets) != 2 {
		t.Errorf("Expected to have 2 fields")
	}
	if len(i.DecimalFacets) != 1 {
		t.Errorf("Expected to 1 number field")
	}
	field, ok := i.KeyFacets[int64(1)]
	if !ok {
		t.Errorf("Expected to have field with id 1, got %v", i.KeyFacets)
	}
	if len(field.Values()) != 2 {
		t.Errorf("Expected to have 2 values in field 1, got %v", field.Values())
	}
}

func TestMultipleIndexMatch(t *testing.T) {
	i := CreateIndex()
	query := Filters{
		StringFilter: []StringSearch{{Id: 1, Value: "test"}},
		NumberFilter: []NumberSearch{{Id: 3, Min: 1, Max: 2}},
		BoolFilter:   []BoolSearch{},
	}
	matching := i.Match(&query)
	if !reflect.DeepEqual(matching, []int64{1, 2}) {
		t.Errorf("Expected [1,2] but got %v", matching)
	}
}

func TestGetMatchItems(t *testing.T) {
	i := CreateIndex()
	query := Filters{
		StringFilter: []StringSearch{{Id: 1, Value: "test"}},
		NumberFilter: []NumberSearch{{Id: 3, Min: 1, Max: 2}},
		BoolFilter:   []BoolSearch{},
	}
	matching := i.Match(&query)
	items := i.GetItems(matching.Ids(), 0, 10)
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
	query := Filters{
		StringFilter: []StringSearch{{Id: 1, Value: "test"}},
		NumberFilter: []NumberSearch{{Id: 3, Min: 1, Max: 2}},
		BoolFilter:   []BoolSearch{},
	}
	matching := i.Match(&query)
	facets := i.GetFacetsFromResult(&matching, &query, &facet.SortIndex{1, 2})
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
