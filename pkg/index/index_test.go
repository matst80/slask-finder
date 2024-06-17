package index

import (
	"testing"

	"tornberg.me/facet-search/pkg/facet"
	"tornberg.me/facet-search/pkg/search"
)

func matchAll(list facet.IdList, ids ...uint) bool {
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

func (l *LoggingChangeHandler) ItemAdded(item *DataItem) {
	l.Printf("Item added %v", *item)
}

func (l *LoggingChangeHandler) ItemChanged(item *DataItem) {
	l.Printf("Item changed %v", *item)
}

func (l *LoggingChangeHandler) ItemDeleted(id uint) {
	l.Printf("Item deleted %v", id)
}

func TestIndexMatch(t *testing.T) {
	i := NewIndex(freetext_search)
	i.AddKeyField(&facet.BaseField{Id: 1, Name: "first", Description: "first field"})
	i.AddKeyField(&facet.BaseField{Id: 2, Name: "other", Description: "other field"})
	i.AddDecimalField(&facet.BaseField{Id: 3, Name: "number", Description: "number field"})
	item := &DataItem{
		BaseItem: BaseItem{
			Id: 1,
		},
		Fields: &[]KeyFieldValue{
			{Value: "test", Id: 1},
			{Value: "hej", Id: 2},
		},
		DecimalFields: &[]NumberFieldValue[float64]{
			{Value: 1, Id: 3},
		},
	}
	i.UpsertItem(item)
	query := Filters{
		StringFilter:  []StringSearch{{Id: 1, Value: "test"}},
		NumberFilter:  []NumberSearch[float64]{{Id: 3, Min: 1, Max: 2}},
		IntegerFilter: []NumberSearch[int]{},
	}
	matching := i.Match(&query)
	if !matchAll(*matching, 1) {
		t.Errorf("Expected [1] but got %v", matching)
	}
}

var token = search.Tokenizer{MaxTokens: 128}
var freetext_search = search.NewFreeTextIndex(&token)

func CreateIndex() *Index {

	i := NewIndex(freetext_search)
	i.AddKeyField(&facet.BaseField{Id: 1, Name: "first", Description: "first field"})
	i.AddKeyField(&facet.BaseField{Id: 2, Name: "other", Description: "other field"})
	i.AddDecimalField(&facet.BaseField{Id: 3, Name: "number", Description: "number field"})

	i.UpsertItem(&DataItem{
		BaseItem: BaseItem{
			Id:    1,
			Title: "item1",
		},
		Fields: &[]KeyFieldValue{
			{Value: "test", Id: 1},
			{Value: "hej", Id: 2},
		},
		DecimalFields: &[]NumberFieldValue[float64]{
			{Value: 1, Id: 3},
		},
	})
	i.UpsertItem(&DataItem{
		BaseItem: BaseItem{
			Id:    2,
			Title: "item2",
		},
		Fields: &[]KeyFieldValue{
			{Value: "test", Id: 1},
			{Value: "slask", Id: 2},
		},
		DecimalFields: &[]NumberFieldValue[float64]{
			{Value: 2, Id: 3},
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
	field, ok := i.KeyFacets[uint(2)]
	if !ok {
		t.Errorf("Expected to have field with id 1, got %v", i.KeyFacets)
	}
	if field.TotalCount() != 2 {
		t.Errorf("Expected to have 2 values in field 1, got %v", field.TotalCount())
	}
}

func TestMultipleIndexMatch(t *testing.T) {
	i := CreateIndex()
	query := Filters{
		StringFilter: []StringSearch{{Id: 1, Value: "test"}},
		NumberFilter: []NumberSearch[float64]{{Id: 3, Min: 1, Max: 2}},
	}
	matching := i.Match(&query)
	if !matchAll(*matching, 1, 2) {
		t.Errorf("Expected [1,2] but got %v", matching)
	}
}

func TestGetMatchItems(t *testing.T) {
	i := CreateIndex()
	query := Filters{
		StringFilter: []StringSearch{{Id: 1, Value: "test"}},
		NumberFilter: []NumberSearch[float64]{{Id: 3, Min: 1, Max: 2}},
	}
	matching := i.Match(&query)
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
		NumberFilter: []NumberSearch[float64]{{Id: 3, Min: 1, Max: 2}},
	}
	matching := i.Match(&query)
	facets := i.GetFacetsFromResult(matching, &query, &facet.SortIndex{1, 2, 3})
	if len(facets.Fields) != 2 {
		t.Errorf("Expected 2 fields but got %v", facets.Fields)
	}
	if len(facets.NumberFields) != 1 {
		t.Errorf("Expected 1 number fields but got %v", facets.NumberFields)
	}
}

func TestUpdateItem(t *testing.T) {
	i := CreateIndex()

	i.ChangeHandler = &LoggingChangeHandler{
		Printf: t.Logf,
	}
	item := &DataItem{
		BaseItem: BaseItem{
			Id: 1,
		},
		Fields: &[]KeyFieldValue{
			{Value: "test", Id: 1},
			{Value: "hej", Id: 2},
		},
		DecimalFields: &[]NumberFieldValue[float64]{
			{Value: 999.0, Id: 3},
		},
	}
	i.UpsertItem(item)
	if (*i.Items[1].Fields)[0].Value != "test" {
		t.Errorf("Expected field 1 to be test")
	}
	if (*i.Items[1].DecimalFields)[0].Value != 999 {
		t.Errorf("Expected field 3 to be 999")
	}

	search := Filters{
		StringFilter:  []StringSearch{},
		NumberFilter:  []NumberSearch[float64]{{Id: 3, Min: 1, Max: 1000}},
		IntegerFilter: []NumberSearch[int]{},
	}
	ids := *i.Match(&search)
	if !matchAll(ids, 1, 2) {
		t.Errorf("Expected 1 ids but got %v", ids)
	}
}

func TestDeleteItem(t *testing.T) {
	i := CreateIndex()
	i.DeleteItem(1)
	if i.HasItem(1) {
		t.Errorf("Expected to not have item with id 1")
	}
	search := Filters{
		StringFilter:  []StringSearch{},
		NumberFilter:  []NumberSearch[float64]{{Id: 3, Min: 1, Max: 1000}},
		IntegerFilter: []NumberSearch[int]{},
	}
	ids := *i.Match(&search)
	if matchAll(ids, 1) {
		t.Errorf("Expected 1 ids but got %v", ids)
	}
}
