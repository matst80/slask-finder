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
	keys   map[string]int
	values []IdList
	len    int
}

func (f *KeyField) Matches(value string) IdList {
	if value == "" {
		return IdList{}
	}
	ret := IdList{}
	idx, found := f.keys[value]
	if found {
		maps.Copy(ret, f.values[idx])
	}
	return ret

}

func (f *KeyField) AddValueLink(value string, id uint) {
	idx, found := f.keys[value]
	if !found {
		f.values = append(f.values, IdList{id: struct{}{}})
		f.keys[value] = f.len
		f.len++
	} else {
		f.values[idx][id] = struct{}{}
	}

	// idList, ok := f.values[value]
	// if !ok {
	// 	if f.values == nil {
	// 		f.values = map[string]IdList{value: {id: fields}}
	// 	} else {
	// 		f.values[value] = IdList{id: fields}
	// 	}
	// } else {
	// 	idList[id] = fields
	// }
}

func (f *KeyField) RemoveValueLink(value string, id uint) {
	idx, found := f.keys[value]
	if found {
		delete(f.values[idx], id)
	}
	// idList, ok := f.values[value]
	// if ok {
	// 	delete(idList, id)
	// }
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

// func (f *KeyField) GetValues() map[string]IdList {
// 	return f.values
// }

// func count(ids IdList, other IdList) int {
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

func NewKeyField(field *BaseField, value string, ids IdList) *KeyField {
	return &KeyField{
		BaseField: field,
		keys:      map[string]int{value: 0},
		len:       1,
		values:    []IdList{ids},
	}
}

func EmptyKeyValueField(field *BaseField) *KeyField {
	return &KeyField{
		BaseField: field,
		keys:      map[string]int{},
		values:    []IdList{},
	}
}
