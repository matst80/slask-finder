package types

import (
	"iter"
	"sync"
)

type StorageProvider interface {
	SaveItems(items iter.Seq[Item]) error
	LoadItems(wg *sync.WaitGroup, handlers ...ItemHandler) error
	SaveGzippedJson(data any, filename string) error
	LoadGzippedJson(data any, filename string) error
	SaveJson(data any, filename string) error
	LoadJson(data any, filename string) error
	SaveGzippedGob(embeddings any, filename string) error
	LoadGzippedGob(output any, filename string) error
}
