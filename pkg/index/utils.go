package index

import "tornberg.me/facet-search/pkg/facet"

func stringValue(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func mapToSliceRef[V BoolResult | StringResult | NumberResult](fields map[int64]*V, sortIndex facet.SortIndex) []*V {

	l := min(len(fields), 64)
	sorted := make([]*V, len(fields))

	idx := 0

	for _, id := range sortIndex {
		if idx >= l {
			break
		}
		f, ok := fields[id]
		if ok {
			sorted[idx] = f

			idx++

		}
	}
	return sorted[:idx]

}

func mapToSlice[V BoolResult | StringResult | NumberResult](fields map[int64]V, sortIndex facet.SortIndex) []V {

	l := min(len(fields), 64)
	sorted := make([]V, len(fields))

	idx := 0

	for _, id := range sortIndex {
		if idx >= l {
			break
		}
		f, ok := fields[id]
		if ok {
			sorted[idx] = f

			idx++

		}
	}
	return sorted[:idx]
}
