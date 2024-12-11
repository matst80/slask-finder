package facet

import (
	"github.com/matst80/slask-finder/pkg/types"
)

type KeyField struct {
	*types.BaseField
	Keys map[string]types.ItemList
}

func (f KeyField) GetType() uint {
	return types.FacetKeyType
}

func (f KeyField) Len() int {
	return len(f.Keys)
}

func (f KeyField) GetValues() []interface{} {
	ret := make([]interface{}, len(f.Keys))
	idx := 0
	for value := range f.Keys {
		ret[idx] = value
		idx++
	}
	return ret
}

func (f *KeyField) match(value string) *types.ItemList {

	ids, ok := f.Keys[value]
	if ok {
		return &ids
	}

	return nil
}

func (f KeyField) Match(input interface{}) *types.ItemList {
	switch val := input.(type) {
	case string:
		return f.match(val)
	case []string:
		ret := make(types.ItemList)
		for _, v := range val {
			r := f.match(v)

			if r != nil {
				ret.Merge(r)
			}

		}
		return &ret
	}

	return &types.ItemList{}
}

func (f KeyField) GetBaseField() *types.BaseField {
	return f.BaseField
}

func (f KeyField) AddValueLink(data interface{}, item types.Item) bool {
	str, ok := data.(string)
	if !ok || str == "" {
		return false
	}
	if len(str) > 64 {
		str = str[:61] + "..."
	}
	itemId := item.GetId()
	if k, ok := f.Keys[str]; ok {
		k.AddId(itemId)
	} else {
		f.Keys[str] = types.ItemList{itemId: struct{}{}}
	}
	return true
}

func (f KeyField) RemoveValueLink(data interface{}, id uint) {
	if str, ok := data.(string); ok {
		if keyId, ok := f.Keys[str]; ok {
			delete(keyId, id)
		}
	}
}

func (f *KeyField) TotalCount() int {
	total := 0
	for _, ids := range f.Keys {
		total += len(ids)
	}
	return total
}

func (f *KeyField) UniqueCount() int {
	return len(f.Keys)
}

func EmptyKeyValueField(field *types.BaseField) KeyField {
	return KeyField{
		BaseField: field,
		Keys:      map[string]types.ItemList{},
	}
}
