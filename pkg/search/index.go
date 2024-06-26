package search

import (
	"sort"
	"sync"

	"tornberg.me/facet-search/pkg/facet"
)

type FreeTextIndex struct {
	mu          sync.Mutex
	Tokenizer   *Tokenizer
	Documents   map[uint]*Document
	TokenMap    map[Token][]*Document
	BaseSortMap map[uint]float64
}

type DocumentResult map[uint]float64

func (i *FreeTextIndex) AddDocument(doc *Document) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.Documents[doc.Id] = doc
	for _, token := range doc.Tokens {
		if _, ok := i.TokenMap[token]; !ok {
			i.TokenMap[token] = make([]*Document, 0)
		}
		i.TokenMap[token] = append(i.TokenMap[token], doc)
	}
}

func (i *FreeTextIndex) CreateDocument(id uint, text ...string) {
	i.AddDocument(i.Tokenizer.MakeDocument(id, text...))
}

func (i *FreeTextIndex) RemoveDocument(id uint) {
	delete(i.Documents, id)
}

func NewFreeTextIndex(tokenizer *Tokenizer) *FreeTextIndex {
	return &FreeTextIndex{
		Tokenizer: tokenizer,
		Documents: make(map[uint]*Document),
		TokenMap:  map[Token][]*Document{},
	}
}

func (i *FreeTextIndex) getMatchDocs(tokens []Token) map[uint]*Document {

	res := make(map[uint]*Document)
	for _, token := range tokens {
		if docs, ok := i.TokenMap[token]; ok {
			for _, doc := range docs {
				res[doc.Id] = doc
			}
		}
	}
	return res
}

func (i *FreeTextIndex) Search(query string) *DocumentResult {
	//i.mu.Lock()
	//defer i.mu.Unlock()
	tokens := i.Tokenizer.Tokenize(query)
	res := make(DocumentResult)

	for _, doc := range i.getMatchDocs(tokens) {
		lastCorrect := 100.0
		for _, t := range doc.Tokens {
			for _, token := range tokens {

				if t == token {
					// Add to result
					res[doc.Id] += lastCorrect
					lastCorrect += 1000
					break
				} else {
					lastCorrect -= 20
				}
			}

		}
		if res[doc.Id] > 0 {
			l := float64(len(tokens))
			dl := float64(len(doc.Tokens))
			base := 0.0
			if i.BaseSortMap != nil {
				if v, ok := i.BaseSortMap[doc.Id]; ok {
					base = v
				}
			}
			hits := res[doc.Id]
			res[doc.Id] = base + ((hits * 10000.0) - ((l - dl) * 100.0))
		}
	}

	return &res
}

func (d *DocumentResult) ToSortIndex() facet.SortIndex {
	l := len(*d)

	sortMap := make(facet.ByValue, l)
	idx := 0
	for id, score := range *d {
		sortMap[idx] = facet.Lookup{Id: id, Value: score}
		idx++
	}
	sort.Sort(sort.Reverse(sortMap))
	sortIndex := make(facet.SortIndex, l)
	for idx, item := range sortMap {
		sortIndex[idx] = item.Id
	}
	return sortIndex
}

type ResultWithSort struct {
	*facet.IdList
	SortIndex facet.SortIndex
}

func (d *DocumentResult) ToResult() *facet.IdList {
	res := facet.IdList{}

	for id := range *d {
		res[id] = struct{}{}
		//res.AddId(id)
	}
	return &res
}

func (d *DocumentResult) ToResultWithSort() ResultWithSort {
	return ResultWithSort{
		IdList:    d.ToResult(),
		SortIndex: d.ToSortIndex(),
	}
}
