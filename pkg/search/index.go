package search

import (
	"log"
	"time"
)

type FreeTextIndex struct {
	Tokenizer Tokenizer
	Documents map[int64]Document
}

type DocumentResult map[int64]int

func (i *FreeTextIndex) AddDocument(doc Document) {
	i.Documents[doc.Id] = doc
}

func (i *FreeTextIndex) RemoveDocument(id int64) {
	delete(i.Documents, id)
}

func NewFreeTextIndex(tokenizer Tokenizer) *FreeTextIndex {
	return &FreeTextIndex{
		Tokenizer: tokenizer,
		Documents: make(map[int64]Document),
	}
}

func (i *FreeTextIndex) Search(query []Token) DocumentResult {
	start := time.Now()
	res := make(DocumentResult)
	for _, doc := range i.Documents {
		for _, token := range query {
			for _, t := range doc.Tokens {
				if t == token {
					// Add to result
					res[doc.Id]++
				}
			}
		}
	}
	log.Printf("Search took %v", time.Since(start))
	return res
}
