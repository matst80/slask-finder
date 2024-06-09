package facet

type FieldNumberValue interface {
	int | float64
}

type FieldKeyValue interface {
	string | bool
}

type FieldValue interface {
	FieldNumberValue | FieldKeyValue
}

type IdList map[int64]struct{}

func (i IdList) Merge(other IdList) {
	for id := range other {
		i[id] = struct{}{}
	}
}

type Field[V FieldValue] struct {
	Id          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	values      map[V]IdList
}

func (f *Field[V]) Matches(value V) Result {

	result := NewResult()

	for key, ids := range f.values {
		if key == value {
			result.Add(ids)
		}
	}

	return result
}

func (f *Field[V]) Values() []V {
	values := make([]V, len(f.values))

	i := 0
	for k := range f.values {
		values[i] = k
		i++
	}
	return values
}

func (f *Field[V]) AddValueLink(value V, id int64) {

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

func (f *Field[V]) TotalCount() int {
	total := 0
	for _, ids := range f.values {
		total += len(ids)
	}
	return total
}

func NewField[V int | float64 | string | bool](field *Field[V], value V, ids IdList) *Field[V] {
	field.values = map[V]IdList{value: ids}
	return field
}

func EmptyValueField[V int | float64 | string | bool](field *Field[V]) *Field[V] {
	field.values = map[V]IdList{}
	return field
}
