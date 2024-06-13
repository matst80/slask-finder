package persistance

import (
	"tornberg.me/facet-search/pkg/index"
	"tornberg.me/facet-search/pkg/search"
)

type Persistance struct {
	File         string
	FreeTextFile string
}

type IndexStorage struct {
	Items map[uint]index.DataItem
}

type FreeTextStorage struct {
	Documents map[uint]search.Document
}
