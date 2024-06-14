package facet

import (
	"maps"
)

type BaseField struct {
	Id          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	HideFacet   bool   `json:"-"`
}

type KeyField struct {
	*BaseField
	values map[string]IdList
}

func (f *KeyField) Matches(value string) IdList {
	if value == "" {
		return IdList{}
	}
	ret := IdList{}

	for key, ids := range f.values {
		if key == value {
			maps.Copy(ret, ids)
		}
	}

	return ret

}

func (f *KeyField) AddValueLink(value string, id uint) {

	idList, ok := f.values[value]
	if !ok {
		if f.values == nil {
			f.values = map[string]IdList{}
		}
		f.values[value] = IdList{id: struct{}{}}
	} else {
		idList[id] = struct{}{}
	}
}

func (f *KeyField) RemoveValueLink(value string, id uint) {
	idList, ok := f.values[value]
	if ok {
		delete(idList, id)
	}
}

func (f *KeyField) TotalCount() int {
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

func (f *KeyField) GetValuesForIds(ids IdList) map[string]int {
	res := map[string]int{}
	for value, valueIds := range f.values {
		idCount := count(valueIds, ids)
		if idCount > 0 {
			res[value] = idCount
		}
	}
	return res
}

func NewKeyField(field *BaseField, value string, ids IdList) *KeyField {
	return &KeyField{
		BaseField: field,
		values:    map[string]IdList{value: ids},
	}
}

func EmptyKeyValueField(field *BaseField) *KeyField {
	return &KeyField{
		BaseField: field,
		values:    map[string]IdList{},
	}
}
