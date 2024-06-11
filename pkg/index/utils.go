package index

import "tornberg.me/facet-search/pkg/facet"

func stringValue(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func boolToString(input map[bool]int) map[string]int {
	result := make(map[string]int)
	for k, v := range input {
		result[stringValue(k)] = v
	}
	return result
}

func boolToStringResult(fields map[int64]*KeyResult[bool]) map[int64]*KeyResult[string] {
	result := make(map[int64]*KeyResult[string])
	for k, v := range fields {

		result[k] = &KeyResult[string]{
			BaseField: v.BaseField,
			Values:    boolToString(v.Values),
		}
	}
	return result
}

func mapToSlice(fields map[int64]*KeyResult[string], sortIndex *facet.SortIndex) []KeyResult[string] {
	l := min(len(fields), 64)
	sorted := make([]KeyResult[string], len(fields))
	idx := 0
	for _, id := range *sortIndex {
		f, ok := fields[id]
		if ok {
			sorted[idx] = *f
			idx++
			if idx >= l {
				break
			}
		}
	}
	return sorted[:idx]
}

func mapToSliceNumber[K float64 | int](fields map[int64]*NumberResult[K], sortIndex *facet.SortIndex) []NumberResult[K] {
	l := min(len(fields), 64)
	sorted := make([]NumberResult[K], len(fields))
	idx := 0
	for _, id := range *sortIndex {
		f, ok := fields[id]
		if ok {
			sorted[idx] = *f
			idx++
			if idx >= l {
				break
			}
		}
	}
	return sorted[:idx]
}
