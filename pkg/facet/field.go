package facet

import (
	"maps"
)

type FieldKeyValue interface {
	string | bool
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

type KeyMatchData[V FieldKeyValue] struct {
	IdList
	FieldId int64
	Value   V
}

func (f *KeyField[V]) Matches(value V) IdList {

	ret := IdList{}

	for key, ids := range f.values {
		if key == value {
			maps.Copy(ret, ids)
		}
	}

	return ret

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

func count(ids IdList, other IdList) int {
	count := 0
	for id := range ids {
		if _, ok := other[id]; ok {
			count++
		}
	}
	return count
}

func (f *KeyField[V]) GetValuesForIds(ids IdList) map[V]int {
	res := map[V]int{}
	for value, valueIds := range f.values {
		idCount := count(valueIds, ids)
		if idCount > 0 {
			res[value] = idCount
		}
	}
	return res
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
