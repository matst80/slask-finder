package facet

import (
	"fmt"
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

func (f KeyField) IsExcludedFromFacets() bool {
	return f.BaseField.HideFacet || f.BaseField.InternalOnly
}

func (f KeyField) IsCategory() bool {
	return f.CategoryLevel > 0
}

type ValueWithCount struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

func (f KeyField) GetValues() []any {
	ret := make([]any, len(f.Keys))
	idx := 0
	for value := range f.Keys {
		ret[idx] = ValueWithCount{
			Value: value,
			Count: len(f.Keys[value]),
		}
		idx++
	}
	return ret
}

func (f *KeyField) match(value string) *types.ItemList {
	if value == "!nil" {
		ret := &types.ItemList{}
		for v, ids := range f.Keys {
			if v == "" {
				continue
			}
			ret.Merge(&ids)
		}
		return ret
	}
	ids, ok := f.Keys[value]
	if ok {
		return &ids
	}

	return &types.ItemList{}
}

func (f KeyField) UpdateBaseField(field *types.BaseField) {
	f.BaseField.UpdateFrom(field)
}

func (f *KeyField) MatchFilterValue(value types.StringFilterValue) *types.ItemList {
	// todo implement
	if value == nil {
		return &types.ItemList{}
	}
	ret := make(types.ItemList)
	for _, v := range value {
		r := f.match(v)

		if r != nil {
			ret.Merge(r)
		}

	}
	return &ret
}

func (f KeyField) Match(input any) *types.ItemList {
	value, ok := types.AsKeyFilterValue(input)
	if !ok {
		log.Printf("KeyField: Match: Unknown type %T", input)
		return &types.ItemList{}
	}
	return f.MatchFilterValue(value)
	// if input == nil {
	// 	return &types.ItemList{}
	// }
	// switch val := input.(type) {
	// case string:
	// 	return f.match(val)
	// case []interface{}:
	// 	ret := make(types.ItemList)
	// 	for _, v := range val {
	// 		if str, ok := v.(string); ok {
	// 			r := f.match(str)
	// 			if r != nil {
	// 				ret.Merge(r)
	// 			}
	// 		}
	// 	}
	// 	return &ret
	// case []string:
	// 	ret := make(types.ItemList)
	// 	for _, v := range val {
	// 		r := f.match(v)

	// 		if r != nil {
	// 			ret.Merge(r)
	// 		}

	// 	}
	// 	return &ret
	// }

	// return &types.ItemList{}
}

// func (f KeyField) MatchAsync(input interface{}, ch chan<- *types.ItemList) {
// 	ch <- f.Match(input)
// }

func (f KeyField) GetBaseField() *types.BaseField {
	return f.BaseField
}

func (f *KeyField) addString(value string, id uint) {
	v := strings.TrimSpace(value)
	if v == "" {
		return
	}
	if f.Type == "stock" {
		if v == "0" {
			return
		}
		v = "Ja"
	} else if f.Type == "bool" {
		low := strings.ToLower(v)
		if low == "no" || low == "nej" || low == "" || low == "false" || low == "x" || low == "saknas" {
			v = "Nej"
		} else {
			v = "Ja"
		}
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

func (f KeyField) AddValueLink(data any, itemId uint) bool {
	if !f.Searchable {
		return false
	}

	switch typed := data.(type) {
	case nil:
		return false
	case float64:
		f.addString(fmt.Sprintf("%f", typed), itemId)
		return true
	case int:
		f.addString(fmt.Sprintf("%d", typed), itemId)
		return true
	case int64:
		f.addString(fmt.Sprintf("%d", typed), itemId)
		return true
	case []string:

		for _, v := range typed {
			f.addString(v, itemId)
		}

		return true
	case string:
		// temporary fix for HTML escaped values, not able index properly
		if strings.Contains(typed, "&lt;") || strings.Contains(typed, "&gt;") {
			log.Printf("KeyField: AddValueLink: Ignoring HTML escaped value, field id: %d", f.Id)
			return false
		}
		parts := strings.Split(typed, ";;")

		for _, partData := range parts {
			f.addString(partData, itemId)
		}

		return true
	// case []any:

	// 	for _, v := range typed {
	// 		if str, ok := v.(string); ok {
	// 			f.addString(str, itemId)
	// 		} else {
	// 			log.Printf("KeyField: AddValueLink: Unknown array type %T, fieldId: %d", v, f.Id)
	// 		}
	// 	}

	// 	return true
	default:
		log.Printf("KeyField: AddValueLink: Unknown type %T, fieldId: %d", typed, f.Id)
	}
	return false
}

func (f KeyField) RemoveValueLink(data any, id uint) {
	switch typed := data.(type) {
	case nil:
		return
	case []any:

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

		parts := strings.Split(typed, ";")

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
