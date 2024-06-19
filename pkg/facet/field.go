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
	values []MatchList
	len    int
}

func (f *KeyField) Matches(value *string) MatchList {
	if *value == "" {
		return MatchList{}
	}
	ret := MatchList{}
	idx, found := f.keys[*value]
	if found {
		maps.Copy(ret, f.values[idx])
	}
	return ret

}

func (f *KeyField) AddValueLink(value string, id uint, fields *ItemFields) {
	idx, found := f.keys[value]
	if !found {
		f.values = append(f.values, MatchList{id: fields})
		f.keys[value] = f.len
		f.len++
	} else {
		f.values[idx][id] = fields
	}
	// idList, ok := f.values[value]
	// if !ok {
	// 	if f.values == nil {
	// 		f.values = map[string]MatchList{value: {id: fields}}
	// 	} else {
	// 		f.values[value] = MatchList{id: fields}
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

// func (f *KeyField) GetValues() map[string]MatchList {
// 	return f.values
// }

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
		keys:      map[string]int{value: 0},
		len:       1,
		values:    []MatchList{ids},
	}
}

func EmptyKeyValueField(field *BaseField) *KeyField {
	return &KeyField{
		BaseField: field,
		keys:      map[string]int{},
		values:    []MatchList{},
	}
}
