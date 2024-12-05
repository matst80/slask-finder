package index

import (
	"github.com/matst80/slask-finder/pkg/search"
	"github.com/matst80/slask-finder/pkg/types"
)

type ContentItem interface {
	GetId() uint
	IndexData() string
}

type ContentIndex struct {
	Items  map[uint]ContentItem
	Search *search.FreeTextIndex
}

func NewContentIndex() *ContentIndex {
	return &ContentIndex{
		Items:  make(map[uint]ContentItem, 0),
		Search: search.NewFreeTextIndex(&search.Tokenizer{MaxTokens: 128}),
	}
}

func (i *ContentIndex) AddItem(item ContentItem) {
	i.Items[item.GetId()] = item
	i.Search.CreateDocument(item.GetId(), item.IndexData())
}

func (i *ContentIndex) MatchQuery(query string) []ContentItem {
	result := i.Search.Search(query)
	sortResult := make(chan *types.SortIndex)
	result.GetSorting(sortResult)
	defer close(sortResult)
	s := <-sortResult
	resultItems := make([]ContentItem, 0, min(20, len(i.Items)))
	j := 0
	for id := range s.SortMap(*result.ToResult()) {
		item, ok := i.Items[id]
		if ok {
			resultItems[j] = item
			j++
		}
		if j >= 20 {
			break
		}
	}
	return resultItems
}
