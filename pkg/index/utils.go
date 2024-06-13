package index

import (
	"hash/fnv"

	"tornberg.me/facet-search/pkg/facet"
)

func (i *Index) mapToSlice(fields map[uint]*KeyResult, sortIndex *facet.SortIndex) []JsonKeyResult {
	l := min(len(fields), 64)
	sorted := make([]JsonKeyResult, len(fields))
	idx := 0
	for _, id := range *sortIndex {
		f, ok := fields[id]
		if ok {
			indexField, baseOk := i.KeyFacets[id]
			if baseOk && !indexField.HideFacet {
				sorted[idx] = JsonKeyResult{
					BaseField: indexField.BaseField,
					Values:    f.GetValues(),
				}
				idx++
				if idx >= l {
					break
				}
			}
		}
	}
	return sorted[:idx]
}

func mapToSliceNumber[K float64 | int](numberFields map[uint]*facet.NumberField[K], fields map[uint]*NumberResult[K], sortIndex *facet.SortIndex) []JsonNumberResult {
	l := min(len(fields), 64)
	sorted := make([]JsonNumberResult, len(fields))
	idx := 0
	for _, id := range *sortIndex {
		f, ok := fields[id]
		if ok {
			indexField, baseOk := numberFields[id]
			if baseOk {

				sorted[idx] = JsonNumberResult{
					BaseField: indexField.BaseField,
					Count:     f.Count,
					Min:       f.Min,
					Max:       f.Max,
				}
				idx++
				if idx >= l {
					break
				}
			}
		}
	}
	return sorted[:idx]
}

func HashString(s string) uint {
	h := fnv.New32a()
	h.Write([]byte(s))
	return uint(h.Sum32())
}
