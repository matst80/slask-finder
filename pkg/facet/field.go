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
	values map[string]MatchList
}

func (f *KeyField) Matches(value string) MatchList {
	if value == "" {
		return MatchList{}
	}
	ret := MatchList{}

	for key, ids := range f.values {
		if key == value {
			maps.Copy(ret, ids)
		}
	}

	return ret

}

func (f *KeyField) AddValueLink(value string, id uint, fields *ItemFields) {
	idList, ok := f.values[value]
	if !ok {
		if f.values == nil {
			f.values = map[string]MatchList{value: {id: fields}}
		} else {
			f.values[value] = MatchList{id: fields}
		}
	} else {
		idList[id] = fields
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

func (f *KeyField) UniqueCount() int {
	return len(f.values)
}

func (f *KeyField) GetValues() map[string]MatchList {
	return f.values
}

// func count(ids MatchList, other IdList) int {
// 	count := 0
// 	for id := range ids {
// 		if _, ok := other[id]; ok {
// 			count++
// 		}
// 	}
// 	return count
// }

// func (f *KeyField) GetValuesForIds(ids IdList) map[string]int {
// 	res := map[string]int{}
// 	for value, valueIds := range f.values {
// 		idCount := count(valueIds, ids)
// 		if idCount > 0 {
// 			res[value] = idCount
// 		}
// 	}
// 	return res
// }

func NewKeyField(field *BaseField, value string, ids MatchList) *KeyField {
	return &KeyField{
		BaseField: field,
		values:    map[string]MatchList{value: ids},
	}
}

func EmptyKeyValueField(field *BaseField) *KeyField {
	return &KeyField{
		BaseField: field,
		values:    map[string]MatchList{},
	}
}
