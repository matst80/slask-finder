package facet

import (
	"log"

	"github.com/matst80/slask-finder/pkg/types"
)

type Tree struct {
	Value    string
	ids      *types.ItemList
	Children map[string]*Tree
}

func NewTree(value string, id types.ItemId) *Tree {
	ids := types.NewItemList()
	ids.AddId(uint32(id))
	return &Tree{
		Value:    value,
		ids:      ids,
		Children: map[string]*Tree{},
	}
}

type TreeField struct {
	*types.BaseField
	Children map[string]*Tree
}

func EmptyTreeValueField(field *types.BaseField) types.Facet {
	return &TreeField{
		BaseField: field,
		Children:  map[string]*Tree{},
	}
}

func (t *TreeField) GetType() uint {
	return types.FacetTreeType
}

func (t *TreeField) match(value []string) *types.ItemList {
	ret := types.NewItemList()
	if len(value) == 0 {
		return ret
	}
	v := value[0]
	curr, ok := t.Children[v]
	if !ok {
		return ret
	} else {
		ret = curr.ids.Clone()
	}
	for _, val := range value[1:] {
		curr, ok = curr.Children[val]
		if !ok {
			return types.NewItemList()
		} else {
			ret = curr.ids.Clone()
		}
	}
	return ret
}

func (t *TreeField) Match(data any) *types.ItemList {
	switch v := data.(type) {
	case []string:
		return t.match(v)
	case []interface{}:
		arr := make([]string, 0, len(v))
		for _, val := range v {
			switch s := val.(type) {
			case string:
				arr = append(arr, s)
			}
		}

		return t.match(arr)
	default:
		log.Printf("tree match value type: %T", data)
	}
	return types.NewItemList()
}

func (t *TreeField) GetBaseField() *types.BaseField {
	return t.BaseField
}

func (t *TreeField) addValue(value []string, id types.ItemId) bool {
	if len(value) == 0 {
		return false
	}
	v := value[0]
	curr, ok := t.Children[v]
	if !ok {
		newTree := NewTree(v, id)
		t.Children[v] = newTree
		curr = newTree
	} else {
		curr.ids.AddId(uint32(id))
	}
	for _, val := range value[1:] {
		f, ok := curr.Children[val]
		if !ok {
			newTree := NewTree(val, id)
			curr.Children[val] = newTree
			curr = newTree
		} else {
			f.ids.AddId(uint32(id))
			curr = f
		}
	}
	return true
}

func (t *TreeField) AddValueLink(value any, id types.ItemId) bool {
	switch v := value.(type) {
	case []string:
		return t.addValue(v, id)
	case []interface{}:
		arr := make([]string, 0, len(v))
		for _, val := range v {
			switch s := val.(type) {
			case string:
				arr = append(arr, s)
			}
		}
		if len(arr) == 0 {
			return false
		}
		return t.addValue(arr, id)
	}
	return false
}

func (t *TreeField) RemoveValueLink(value any, id types.ItemId) {
}

func (t *TreeField) UpdateBaseField(data *types.BaseField) {
	t.BaseField.UpdateFrom(data)
}

func (t *TreeField) GetValues() []any {
	return nil
}

func (t *TreeField) IsExcludedFromFacets() bool {
	return false
}

func (t *TreeField) IsCategory() bool {
	return true
}
