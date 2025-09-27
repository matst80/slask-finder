package facet

import (
	"hash/fnv"

	"github.com/matst80/slask-finder/pkg/types"
)

func HashString(s string) uint {
	h := fnv.New32a()
	h.Write([]byte(s))
	return uint(h.Sum32())
}

type FieldType uint

type StorageFacet struct {
	*types.BaseField
	Type FieldType `json:"type"`
}
