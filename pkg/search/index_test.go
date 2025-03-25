package search

import "testing"

func TestDocumentIndex(t *testing.T) {
	token := Tokenizer{
		MaxTokens: 100,
	}
	idx := NewFreeTextIndex(&token)
	idx.AddDocument(token.MakeDocument(1, "Hello world, how are you?", "Other property"))
	idx.AddDocument(token.MakeDocument(2, "Hello slask, how are world?", "Some other text"))
	idx.AddDocument(token.MakeDocument(3, "Hello slask, how are you?", "Some other text"))

	res := idx.Search("Hello world")
	if len(*res) != 3 {
		t.Errorf("Expected 3 results but got %d", len(*res))
	}
	sort := *res
	if len(sort) != 3 {
		t.Errorf("Expected 3 results but got %d", len(sort))
	}
	// if sort[0] != 1 {
	// 	t.Errorf("Expected hello world: 1 to be first, %d", sort)
	// }
	// if sort[1] != 2 {
	// 	t.Errorf("Expected hello slask ... world to be second, %d", sort)
	// }
	// if sort[2] != 3 {
	// 	t.Errorf("Expected hello slask ... world to be second, %d", sort)
	// }
}

func TestDocument2Index(t *testing.T) {
	token := Tokenizer{
		MaxTokens: 100,
	}
	idx := NewFreeTextIndex(&token)
	idx.AddDocument(token.MakeDocument(1, "Other property", "9900X3D"))
	idx.AddDocument(token.MakeDocument(2, "Some other text slask", "AMD 9600X3D"))
	idx.AddDocument(token.MakeDocument(3, "Hello slask, how are you?", "Some other text"))

	res := idx.Search("x3d")
	if len(*res) != 2 {
		t.Errorf("Expected 2 results but got %d", len(*res))
	}

	// if sort[0] != 1 {
	// 	t.Errorf("Expected hello world: 1 to be first, %d", sort)
	// }
	// if sort[1] != 2 {
	// 	t.Errorf("Expected hello slask ... world to be second, %d", sort)
	// }
	// if sort[2] != 3 {
	// 	t.Errorf("Expected hello slask ... world to be second, %d", sort)
	// }
}
