package persistance

import (
	"tornberg.me/facet-search/pkg/index"
	"tornberg.me/facet-search/pkg/search"
)

type GobPersistance struct {
	File         string
	FreeTextFile string
}

type FilePersistance interface {
	Save(data any) error
	Load(data *any) error
}

type IndexPersistance interface {
	LoadIndex(idx *index.Index) error
	SaveIndex(idx *index.Index) error
}

type FreeTextPersistance interface {
	LoadFreeText(ft *search.FreeTextIndex) error
	SaveFreeText(ft *search.FreeTextIndex) error
}

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
