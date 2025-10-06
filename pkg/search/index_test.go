package search

import "testing"

func TestDocumentIndex(t *testing.T) {

	idx := NewFreeTextItemHandler(DefaultFreeTextHandlerOptions())
	idx.CreateDocumentUnsafe(1, "Hello world, how are you?", "Other property")
	idx.CreateDocumentUnsafe(2, "Hello slask, how are world?", "Some other text")
	idx.CreateDocumentUnsafe(3, "Hello slask, how are you?", "Some other text")

	res := idx.Search("Hello world")
	if res.Bitmap().GetCardinality() != 3 {
		t.Errorf("Expected 3 results but got %d", res.Bitmap().GetCardinality())
	}

}

func TestDocument2Index(t *testing.T) {

	idx := NewFreeTextItemHandler(DefaultFreeTextHandlerOptions())
	idx.CreateDocumentUnsafe(1, "Other property", "9900X3D")
	idx.CreateDocumentUnsafe(2, "Some other text slask", "AMD 9600X3D")
	idx.CreateDocumentUnsafe(3, "Hello slask, how are you?", "Some other text")

	res := idx.Search("x3d")
	if res.Bitmap().GetCardinality() != 2 {
		t.Errorf("Expected 2 results but got %d", res.Bitmap().GetCardinality())
	}

}
