package facet

import (
	"log"
	"strings"

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
	if value == "!nil" {
		ret := make(types.ItemList)
		for v, ids := range f.Keys {
			if v == "" {
				continue
			}
			ret.Merge(&ids)
		}
		return &ret
	}
	ids, ok := f.Keys[value]
	if ok {
		return &ids
	}

	return nil
}

func (f KeyField) Match(input interface{}) *types.ItemList {
	if input == nil {
		return &types.ItemList{}
	}
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

func (f KeyField) MatchAsync(input interface{}, ch chan<- *types.ItemList) {
	ch <- f.Match(input)
}

func (f KeyField) GetBaseField() *types.BaseField {
	return f.BaseField
}

func (f *KeyField) addString(value string, id uint) {
	v := strings.TrimSpace(value)
	if v == "" {
		return
	}

	if k, ok := f.Keys[v]; ok {
		k.AddId(id)
	} else {
		f.Keys[v] = types.ItemList{id: struct{}{}}
	}

}

func (f *KeyField) removeString(value string, id uint) {
	v := strings.TrimSpace(value)
	if v == "" {
		return
	}

	if k, ok := f.Keys[v]; ok {
		delete(k, id)
		if len(k) == 0 {
			delete(f.Keys, v)
		}
	}
}

func (f KeyField) AddValueLink(data interface{}, itemId uint) bool {
	if !f.Searchable {
		return false
	}

	switch typed := data.(type) {
	case nil:
		return false
	case []interface{}:

		for _, v := range typed {
			if str, ok := v.(string); ok {
				f.addString(str, itemId)
			}
		}
		return true
	case []string:

		for _, v := range typed {
			f.addString(v, itemId)
		}

		return true
	case string:

		if strings.Contains(typed, "&lt;") || strings.Contains(typed, "&gt;") {
			return false
		}
		parts := strings.Split(typed, ";")

		for _, partData := range parts {
			f.addString(partData, itemId)
		}

		return true
	default:
		log.Printf("KeyField: AddValueLink: Unknown type %T, fieldId: %d", typed, f.Id)
	}
	return false
}

func (f KeyField) RemoveValueLink(data interface{}, id uint) {
	switch typed := data.(type) {
	case nil:
		return
	case []interface{}:

		for _, v := range typed {
			if str, ok := v.(string); ok {
				f.removeString(str, id)
			}
		}
		return
	case []string:

		for _, v := range typed {
			f.removeString(v, id)
		}

		return
	case string:

		if strings.Contains(typed, "&lt;") || strings.Contains(typed, "&gt;") {
			log.Printf("KeyField: RemoveValueLink: Invalid string %s, facetid: %d", typed, f.Id)
			return
		}
		parts := strings.Split(typed, ";")
		if len(parts) > 1 {
			log.Printf("KeyField: RemoveValueLink: array string %s, facetid: %d", typed, f.Id)
		}

		for _, partData := range parts {
			f.removeString(partData, id)
		}

		return
	default:
		log.Printf("KeyField: AddValueLink: Unknown type %T, fieldId: %d", typed, f.Id)
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
