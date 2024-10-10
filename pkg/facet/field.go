package facet

type BaseField struct {
	Id               uint    `json:"id"`
	Name             string  `json:"name"`
	Description      string  `json:"description,omitempty"`
	Priority         float64 `json:"prio,omitempty"`
	Type             string  `json:"type,omitempty"`
	LinkedId         uint    `json:"linkedId,omitempty"`
	ValueSorting     uint    `json:"sorting,omitempty"`
	HideFacet        bool    `json:"-"`
	CategoryLevel    int     `json:"categoryLevel,omitempty"`
	IgnoreIfInSearch bool    `json:"-"`
}

type KeyField struct {
	*BaseField
	keys map[string]*IdList
	//values []IdList
	//len    uint
}

func (f *KeyField) GetValues() []string {
	ret := make([]string, len(f.keys))
	idx := 0
	for value := range f.keys {
		ret[idx] = value
		idx++
	}
	return ret
}

func (f *KeyField) Matches(value string) *IdList {

	list, found := f.keys[value]
	if found {
		return list
	}
	return &IdList{}

}

func (f *KeyField) AddValueLink(value string, id uint) {
	list, found := f.keys[value]
	if !found {
		f.keys[value] = &IdList{id: struct{}{}}
	} else {
		list.Add(id)
		//(*key)[id] = struct{}{}
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
	key, found := f.keys[value]
	if found {
		delete(*key, id)
	}
	// idList, ok := f.values[value]
	// if ok {
	// 	delete(idList, id)
	// }
}

func (f *KeyField) TotalCount() int {
	total := 0
	for _, ids := range f.keys {
		total += len(*ids)
	}
	return total
}

func (f *KeyField) UniqueCount() int {
	return len(f.keys)
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

func NewKeyField(field *BaseField, value string, ids *IdList) *KeyField {
	return &KeyField{
		BaseField: field,
		keys:      map[string]*IdList{value: ids},
	}
}

func EmptyKeyValueField(field *BaseField) *KeyField {
	return &KeyField{
		BaseField: field,
		keys:      map[string]*IdList{},
	}
}
