package facet

type BoolValueField struct {
	Field  `json:"field"`
	values map[bool][]int64
}

func (f *BoolValueField) Matches(value bool) Result {
	result := NewResult()

	for v, ids := range f.values {
		if v == value {
			result.Add(ids...)
		}
	}

	return result
}

func (f *BoolValueField) Values() []bool {
	l := len(f.values)
	if l == 0 {
		return []bool{}
	}
	values := make([]bool, l)
	idx := 0
	for value := range f.values {
		values[idx] = value
		idx++
	}

	return values
}

func (f *BoolValueField) TotalCount() int {
	total := 0
	for _, ids := range f.values {
		total += len(ids)
	}
	return total
}

func (f *BoolValueField) AddValueLink(value bool, ids ...int64) {
	f.values[value] = append(f.values[value], ids...)
}

func NewBoolValueField(field Field, value bool, ids ...int64) BoolValueField {
	return BoolValueField{
		Field:  field,
		values: map[bool][]int64{value: ids},
	}
}

func EmptyBoolField(field Field) BoolValueField {
	return BoolValueField{
		Field:  field,
		values: map[bool][]int64{},
	}
}
