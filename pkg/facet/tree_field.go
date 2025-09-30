package facet

import (
	"log"
	"strings"

	"github.com/matst80/slask-finder/pkg/types"
)

type Branch struct {
	Branches map[string]*Branch `json:"children"`
	items    types.ItemList
}

type TreeField struct {
	*types.BaseField
	*Branch
}

func (f TreeField) GetType() uint {
	return types.FacetTreeType
}

func (f TreeField) Len() int {
	return len(f.Branches)
}

func (f TreeField) GetValues() []any {
	ret := make([]any, len(f.Branches))
	idx := 0
	for key := range f.Branches {
		ret[idx] = key
		idx++
	}
	return ret
}

func (f TreeField) Match(input any) *types.ItemList {
	if input == nil {
		return &types.ItemList{}
	}
	switch val := input.(type) {

	case []string:
		var ret types.ItemList
		current := f.Branch
		for _, v := range val {
			if found, ok := current.Branches[v]; ok {
				current = found
				ret = current.items
			} else {
				break
			}
		}
		return &ret
	}

	return &types.ItemList{}
}

func (f TreeField) MatchAsync(input any, ch chan<- *types.ItemList) {
	ch <- f.Match(input)
}

func (f TreeField) GetBaseField() *types.BaseField {
	return f.BaseField
}

func (f *TreeField) addValue(keys []string, id uint) {
	current := f.Branch
	branches := make([]*Branch, len(keys))
	for i, key := range keys {
		if t, ok := current.Branches[key]; !ok {
			b := &Branch{
				Branches: map[string]*Branch{},
				items:    types.ItemList{},
			}
			current.Branches[key] = b
			branches[i] = b
		} else {
			current = t
			branches[i] = t
		}

	}
	for _, b := range branches {
		b.items.AddId(id)
	}
}

func (f TreeField) AddValueLink(data any, item types.Item) bool {
	if !f.Searchable {
		return false
	}
	switch typed := data.(type) {
	case nil:
		return false
	case []any:
		itemId := item.GetId()
		keys := make([]string, 0)
		for _, v := range typed {
			if str, ok := v.(string); ok {
				part := strings.TrimSpace(str)
				if part == "" {
					continue
				}
				keys = append(keys, part)
			}
		}
		if len(keys) > 0 {
			f.addValue(keys, itemId)
		}
		return true
	case []string:
		itemId := item.GetId()
		keys := make([]string, 0)
		for _, v := range typed {
			part := strings.TrimSpace(v)
			if part == "" {
				continue
			}

			keys = append(keys, part)
		}
		if len(keys) > 0 {
			f.addValue(keys, itemId)
		}
		return true
	case string:
		key := strings.TrimSpace(typed)
		if key == "" {
			return false
		}
		f.addValue([]string{key}, item.GetId())

		return true
	default:
		log.Printf("TreeField: AddValueLink: Unknown type %T, fieldId: %d", typed, f.Id)
	}
	return false
}

func (f TreeField) RemoveValueLink(data any, id uint) {
	// if str, ok := data.(string); ok {
	// 	if keyId, ok := f.Keys[str]; ok {
	// 		delete(keyId, id)
	// 	}
	// }

}

func (f *TreeField) TotalCount() int {
	total := 0
	for _, ids := range f.Branches {
		total += len(ids.items)
	}
	return total
}

func (f *TreeField) UniqueCount() int {
	return len(f.Branches)
}

func EmptyTreeValueField(field *types.BaseField) TreeField {
	return TreeField{
		BaseField: field,
		Branch: &Branch{
			Branches: map[string]*Branch{},
			items:    types.ItemList{},
		},
	}
}
