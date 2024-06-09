package search

import (
	"log"
	"sort"
	"time"

	"tornberg.me/facet-search/pkg/facet"
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
		if res[doc.Id] > 0 {
			res[doc.Id] = (res[doc.Id] / len(query)) * 100
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
	Result    facet.Result
	SortIndex facet.SortIndex
}

func (d *DocumentResult) ToResult() facet.Result {
	res := facet.NewResult()
	for id := range *d {
		res.Add(id)
	}
	return res
}

func (d *DocumentResult) ToResultWithSort() ResultWithSort {
	return ResultWithSort{
		Result:    d.ToResult(),
		SortIndex: d.ToSortIndex(),
	}
}
