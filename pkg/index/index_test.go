package index

import (
	"testing"

	"github.com/matst80/slask-finder/pkg/facet"
	"github.com/matst80/slask-finder/pkg/search"
	"github.com/matst80/slask-finder/pkg/types"
)

func matchAll(list types.ItemList, ids ...uint) bool {
	for _, id := range ids {
		if _, ok := list[id]; !ok {
			return false
		}
	}
	return true
}

type LoggingChangeHandler struct {
	Printf func(format string, v ...interface{})
}

func (l *LoggingChangeHandler) ItemsUpserted(item []types.Item) {
	l.Printf("Items added %v", len(item))
}

func (l *LoggingChangeHandler) ItemDeleted(id uint) {
	l.Printf("Item deleted %v", id)
}

func (l *LoggingChangeHandler) PriceLowered(item []types.Item) {
	l.Printf("Prices lowered %v", len(item))
}

func TestIndexMatch(t *testing.T) {
	i := NewIndex(freetext_search)
	i.AddKeyField(&types.BaseField{Id: 1, Name: "first", Description: "first field"})
	i.AddKeyField(&types.BaseField{Id: 2, Name: "other", Description: "other field"})
	i.AddDecimalField(&types.BaseField{Id: 3, Name: "number", Description: "number field"})
	item := DataItem{
		BaseItem: &BaseItem{
			Id: 1,
		},
		Fields: types.ItemFields{1: "test", 2: "hej", 3: 1.0},
	}
	i.UpsertItem(&item)
	query := Filters{
		StringFilter:  []StringSearch{{Id: 1, Value: "test"}},
		NumberFilter:  []NumberSearch[float64]{{Id: 3, NumberRange: facet.NumberRange[float64]{Min: 1, Max: 2}}},
		IntegerFilter: []NumberSearch[int]{},
	}
	ch := make(chan *types.ItemList)
	defer close(ch)
	i.Match(&query, nil, ch)
	matching := <-ch
	if !matchAll(*matching, 1) {
		t.Errorf("Expected [1] but got %v", matching)
	}
}

var token = search.Tokenizer{MaxTokens: 128}
var freetext_search = search.NewFreeTextIndex(&token)

func CreateIndex() *Index {

	i := NewIndex(freetext_search)
	i.AddKeyField(&types.BaseField{Id: 1, Name: "first", Description: "first field"})
	i.AddKeyField(&types.BaseField{Id: 2, Name: "other", Description: "other field"})
	i.AddDecimalField(&types.BaseField{Id: 3, Name: "number", Description: "number field"})

	i.UpsertItem(&DataItem{
		BaseItem: &BaseItem{
			Id:    1,
			Title: "item1",
		},
		Fields: types.ItemFields{1: "test", 2: "hej", 3: 1.0},
	})
	i.UpsertItem(&DataItem{
		BaseItem: &BaseItem{
			Id:    2,
			Title: "item2",
		},
		Fields: types.ItemFields{1: "test", 2: "slask", 3: 2.0},
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
	if len(i.Facets) != 3 {
		t.Errorf("Expected to have 2 fields")
	}

	_, ok := i.Facets[uint(2)]
	if !ok {
		t.Errorf("Expected to have field with id 1, got %v", i.Facets)
	}

}

func TestMultipleIndexMatch(t *testing.T) {
	i := CreateIndex()
	query := Filters{
		StringFilter: []StringSearch{{Id: 1, Value: "test"}},
		NumberFilter: []NumberSearch[float64]{{Id: 3, NumberRange: facet.NumberRange[float64]{Min: 1, Max: 2}}},
	}
	ch := make(chan *types.ItemList)
	defer close(ch)
	i.Match(&query, nil, ch)
	matching := <-ch
	if !matchAll(*matching, 1, 2) {
		t.Errorf("Expected [1,2] but got %v", matching)
	}
}

func TestGetMatchItems(t *testing.T) {
	i := CreateIndex()
	query := Filters{
		StringFilter: []StringSearch{{Id: 1, Value: "test"}},
		NumberFilter: []NumberSearch[float64]{{Id: 3, NumberRange: facet.NumberRange[float64]{Min: 1, Max: 2}}},
	}
	ch := make(chan *types.ItemList)
	defer close(ch)
	i.Match(&query, nil, ch)
	matching := <-ch
	//items := i.GetItems(matching, 0, 10)
	if !matchAll(*matching, 1, 2) {
		t.Errorf("Expected ids [1,2] but got %v", matching)
	}
	// if items[0].Title != "item1" || items[1].Title != "item2" {
	// 	t.Errorf("Expected titles [item1, item2] but got %v", items)
	// }
	// if items[0].Fields[0].Value != "test" || items[1].Fields[0].Value != "hoj" {
	// 	t.Errorf("Expected fields [test, hoj] but got %v", items)
	// }
}

func TestGetFacetsFromResultIds(t *testing.T) {
	i := CreateIndex()
	query := Filters{
		StringFilter: []StringSearch{{Id: 1, Value: "test"}},
		NumberFilter: []NumberSearch[float64]{{Id: 3, NumberRange: facet.NumberRange[float64]{Min: 1, Max: 2}}},
	}
	ch := make(chan *types.ItemList)
	defer close(ch)
	i.Match(&query, nil, ch)
	matching := <-ch
	facets := i.GetFacetsFromResult(*matching, &query, &types.SortIndex{1, 2, 3})
	if len(facets) != 3 {
		t.Errorf("Expected 3 fields but got %v", facets)
	}

}

func TestUpdateItem(t *testing.T) {
	i := CreateIndex()

	i.ChangeHandler = &LoggingChangeHandler{
		Printf: t.Logf,
	}
	item := DataItem{
		BaseItem: &BaseItem{
			Id: 1,
		},
		Fields: types.ItemFields{
			1: "test",
			2: "hej",
			3: 999.0,
		},
	}
	i.UpsertItem(&item)
	// if i.Items[1].Facets[0].Value != "test" {
	// 	t.Errorf("Expected field 1 to be test")
	// }
	// if i.Items[1].DecimalFields[0].Value != 999 {
	// 	t.Errorf("Expected field 3 to be 999")
	// }

	// search := Filters{
	// 	StringFilter:  []StringSearch{},
	// 	NumberFilter:  []NumberSearch[float64]{{Id: 3, Min: 1, Max: 1000}},
	// 	IntegerFilter: []NumberSearch[int]{},
	// }
	// ch := make(chan *facet.IdList)
	// defer close(ch)
	// i.Match(&search, nil, ch)
	// ids := <-ch
	// if !matchAll(*ids, 1, 2) {
	// 	t.Errorf("Expected 1 ids but got %v", ids)
	// }
}

func TestDeleteItem(t *testing.T) {
	i := CreateIndex()
	i.DeleteItem(1)
	if i.HasItem(1) {
		t.Errorf("Expected to not have item with id 1")
	}
	search := Filters{
		StringFilter:  []StringSearch{},
		NumberFilter:  []NumberSearch[float64]{{Id: 3, NumberRange: facet.NumberRange[float64]{Min: 1, Max: 1000}}},
		IntegerFilter: []NumberSearch[int]{},
	}
	ch := make(chan *types.ItemList)
	defer close(ch)
	i.Match(&search, nil, ch)
	ids := <-ch
	if matchAll(*ids, 1) {
		t.Errorf("Expected 1 ids but got %v", ids)
	}
}
