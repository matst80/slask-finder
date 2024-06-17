package search

import "testing"

func TestDocumentIndex(t *testing.T) {
	token := Tokenizer{
		MaxTokens: 100,
	}
	idx := NewFreeTextIndex(&token)
	idx.AddDocument(token.MakeDocument(1, "Hello world, how are you?", "Other property"))
	idx.AddDocument(token.MakeDocument(2, "Hello slask, how are world?", "Some other text"))
	idx.AddDocument(token.MakeDocument(2, "Hello slask, how are you?", "Some other text"))

	res := idx.Search("Hello world")
	if len(*res) != 2 {
		t.Errorf("Expected 2 results but got %d", len(*res))
	}
	sort := res.ToSortIndex(nil)
	if len(sort) != 2 {
		t.Errorf("Expected 2 results but got %d", len(sort))
	}
	if sort[0] != 1 {
		t.Errorf("Expected hello world: 1 to be first, %d", sort)
	}
	if sort[1] != 2 {
		t.Errorf("Expected hello slask ... world to be second, %d", sort)
	}
}
