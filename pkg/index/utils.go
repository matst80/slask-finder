package index

import "tornberg.me/facet-search/pkg/facet"

func mapToSlice(fields map[int]KeyResult, sortIndex *facet.SortIndex) []JsonKeyResult {
	l := min(len(fields), 64)
	sorted := make([]JsonKeyResult, len(fields))
	idx := 0
	for _, id := range *sortIndex {
		f, ok := fields[id]
		if ok {
			sorted[idx] = JsonKeyResult{
				BaseField: f.BaseField,
				Values:    f.GetValues(),
			}
			idx++
			if idx >= l {
				break
			}
		}
	}
	return sorted[:idx]
}

func mapToSliceNumber[K float64 | int](fields map[int]NumberResult[K], sortIndex *facet.SortIndex) []NumberResult[K] {
	l := min(len(fields), 64)
	sorted := make([]NumberResult[K], len(fields))
	idx := 0
	for _, id := range *sortIndex {
		f, ok := fields[id]
		if ok {
			sorted[idx] = f
			idx++
			if idx >= l {
				break
			}
		}
	}
	return sorted[:idx]
}
