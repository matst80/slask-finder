package facet

import "hash/fnv"

func HashString(s string) uint {
	h := fnv.New32a()
	h.Write([]byte(s))
	return uint(h.Sum32())
}
