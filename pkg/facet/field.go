package facet

type FieldNumberValue interface {
	int | float64
}

type FieldKeyValue interface {
	string | bool
}

type IdList map[int64]struct{}

func (i IdList) Merge(other IdList) {
	for id := range other {
		i[id] = struct{}{}
	}
}

type BaseField struct {
	Id          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type KeyField[V FieldKeyValue] struct {
	*BaseField
	values map[V]IdList
}

func (f *KeyField[V]) Matches(value V) Result {

	result := NewResult()

	for key, ids := range f.values {
		if key == value {
			result.Add(ids)
		}
	}

	return result
}

func (f *KeyField[V]) Values() []V {
	values := make([]V, len(f.values))

	i := 0
	for k := range f.values {
		values[i] = k
		i++
	}
	return values
}

func (f *KeyField[V]) AddValueLink(value V, id int64) {

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

func (f *KeyField[V]) TotalCount() int {
	total := 0
	for _, ids := range f.values {
		total += len(ids)
	}
	return total
}

func NewKeyField[V FieldKeyValue](field *BaseField, value V, ids IdList) *KeyField[V] {
	return &KeyField[V]{
		BaseField: field,
		values:    map[V]IdList{value: ids},
	}
}

func EmptyKeyValueField[V FieldKeyValue](field *BaseField) *KeyField[V] {
	return &KeyField[V]{
		BaseField: field,
		values:    map[V]IdList{},
	}
}
