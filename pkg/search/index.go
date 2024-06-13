package search

import (
	"log"
	"sort"
	"time"

	"tornberg.me/facet-search/pkg/facet"
)

type FreeTextIndex struct {
	Tokenizer *Tokenizer
	Documents map[uint]*Document
}

type DocumentResult map[uint]float64

func (i *FreeTextIndex) AddDocument(doc *Document) {
	i.Documents[doc.Id] = doc
}

func (i *FreeTextIndex) CreateDocument(id uint, text ...string) {
	i.Documents[id] = i.Tokenizer.MakeDocument(id, text...)
}

func (i *FreeTextIndex) RemoveDocument(id uint) {
	delete(i.Documents, id)
}

func NewFreeTextIndex(tokenizer *Tokenizer) *FreeTextIndex {
	return &FreeTextIndex{
		Tokenizer: tokenizer,
		Documents: make(map[uint]*Document),
	}
}

func (i *FreeTextIndex) Search(query string) DocumentResult {
	tokens := i.Tokenizer.Tokenize(query)
	start := time.Now()
	res := make(DocumentResult)

	for _, doc := range i.Documents {
		lastCorrect := 1.0
		for _, token := range tokens {
			for _, t := range doc.Tokens {
				if t == token {
					// Add to result
					res[doc.Id] += lastCorrect
					lastCorrect *= 2
				} else {
					lastCorrect = 1
				}
			}
		}
		if res[doc.Id] > 0 {
			l := float64(len(tokens))
			dl := float64(len(doc.Tokens))
			res[doc.Id] = ((res[doc.Id] / l) * 1000.0) - (l - dl)
		}
	}
	log.Printf("Search took %v", time.Since(start))
	return res
}

func (d *DocumentResult) ToSortIndex() facet.SortIndex {
	l := len(*d)
	sortIndex := make(facet.SortIndex, l)
	sortMap := make(facet.ByValue, l)
	idx := 0
	for id, score := range *d {
		sortMap[idx] = facet.Lookup{Id: id, Value: float64(score / 100.0)}
		idx++
	}
	sort.Sort(sort.Reverse(sortMap))
	for idx, item := range sortMap {
		sortIndex[idx] = item.Id
	}
	return sortIndex
}

type ResultWithSort struct {
	facet.IdList
	SortIndex facet.SortIndex
}

func (d *DocumentResult) ToResult() facet.IdList {
	res := facet.IdList{}

	for id := range *d {
		res[id] = struct{}{}
		//res.AddId(id)
	}
	return res
}

func (d *DocumentResult) ToResultWithSort() ResultWithSort {
	return ResultWithSort{
		IdList:    d.ToResult(),
		SortIndex: d.ToSortIndex(),
	}
}
