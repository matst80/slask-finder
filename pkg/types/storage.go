package types

import (
	"iter"
)

type StorageProvider interface {
	SaveItems(items iter.Seq[Item]) error
	LoadItems(handlers ...ItemHandler) error
	SaveGzippedJson(data any, filename string) error
	LoadGzippedJson(data interface{}, filename string) error
	SaveJson(data any, filename string) error
	LoadJson(data interface{}, filename string) error
	SaveGzippedGob(embeddings any, filename string) error
	LoadGzippedGob(output interface{}, filename string) error
}
