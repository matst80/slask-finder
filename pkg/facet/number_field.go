package facet

type NumberField[V FieldNumberValue] Field[V]

func (f *NumberField[V]) MatchesRange(min V, max V) Result {
	result := NewResult()

	for v, ids := range f.values {
		if v >= min && v <= max {
			result.Add(ids)
		}
	}

	return result
}

type NumberRange[V FieldNumberValue] struct {
	Min    V   `json:"min"`
	Max    V   `json:"max"`
	Values []V `json:"values"`
}

func (f *NumberField[V]) Bounds() NumberRange[V] {
	values := []V{}
	min := V(99999999999999999)
	max := V(-99999999999999999)
	for value := range f.values {
		if value < min {
			min = value
		}
		if value > max {
			max = value
		}
		values = append(values, value)
	}

	return NumberRange[V]{Min: min, Max: max, Values: values}
}

func (f *NumberField[V]) AddValueLink(value V, id int64) {
	idList, ok := f.values[value]
	if !ok {
		if f.values == nil {
			f.values = map[V]IdList{}
		}
		f.values[value] = IdList{id: struct{}{}}
	} else {
		idList[id] = struct{}{}
	}
}

func (f *NumberField[V]) TotalCount() int {
	total := 0
	for _, ids := range f.values {
		total += len(ids)
	}
	return total
}
