package facet

import (
	"unsafe"

	"tornberg.me/facet-search/pkg/types"
)

type KeyField struct {
	*types.BaseField
	keys map[interface{}]types.ItemList
}

func (f KeyField) GetType() uint {
	return types.FacetKeyType
}

func (f KeyField) Size() int {
	sum := 0
	for key, ids := range f.keys {
		sum += int(unsafe.Sizeof(ids)) + len(key.(string))
	}
	return sum
}

func (f KeyField) GetValues() []interface{} {
	ret := make([]interface{}, len(f.keys))
	idx := 0
	for value := range f.keys {
		ret[idx] = value
		idx++
	}
	return ret
}

func (f KeyField) Match(input interface{}) *types.ItemList {
	//value, ok := input.(string)
	//if ok {

	list, found := f.keys[input]
	if found {
		return &list
	}
	//}
	return &types.ItemList{}
}

func (f KeyField) GetBaseField() *types.BaseField {
	return f.BaseField
}

func (f KeyField) AddValueLink(data interface{}, item types.Item) bool {
	str, ok := data.(string)
	if !ok {
		return false
	}
	if len(str) > 64 {
		data = str[:61] + "..."
	}
	list, found := f.keys[data]
	if !found {
		f.keys[data] = types.ItemList{item.GetId(): struct{}{}}
	} else {
		list.Add(item)
	}
	return true
}

func (f KeyField) RemoveValueLink(data interface{}, id uint) {

	key, found := f.keys[data]
	if found {
		delete(key, id)
	}

}

func (f *KeyField) TotalCount() int {
	total := 0
	for _, ids := range f.keys {
		total += len(ids)
	}
	return total
}

func (f *KeyField) UniqueCount() int {
	return len(f.keys)
}

func EmptyKeyValueField(field *types.BaseField) KeyField {
	return KeyField{
		BaseField: field,
		keys:      map[interface{}]types.ItemList{},
	}
}
