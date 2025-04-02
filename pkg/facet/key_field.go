package facet

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/matst80/slask-finder/pkg/types"
)

type KeyField struct {
	*types.BaseField
	Keys    map[string]types.ItemList
	changed bool
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

func (f KeyField) AddValueLink(data interface{}, item types.Item) bool {
	switch typed := data.(type) {
	case nil:
		return false
	case []string:
		for _, v := range typed {
			if v == "" {
				continue
			}
			if !f.AddValueLink(v, item) {
				return false
			}
		}
		f.setChanged(true)
		return true
	case string:

		if typed == "" {
			return false
		}
		if strings.Contains(typed, "&lt;") || strings.Contains(typed, "&gt;") {
			return false
		}
		parts := strings.Split(typed, ";")
		// if len(parts) > 1 {
		// 	log.Print("found keys", strings.Join(parts, " / "))
		// }
		itemId := item.GetId()
		for _, partData := range parts {
			part := strings.TrimSpace(partData)
			if part == "" {
				continue
			}
			// if len(part) > 128 {
			// 	log.Printf("Truncating key value %s", part)
			// 	part = part[:126] + "..."
			// }

			if k, ok := f.Keys[part]; ok {
				k.AddId(itemId)
			} else {
				f.Keys[part] = types.ItemList{itemId: struct{}{}}
			}
			f.setChanged(true)
		}

		return true
	}
	return false
}

func (f KeyField) RemoveValueLink(data interface{}, id uint) {
	if str, ok := data.(string); ok {
		if keyId, ok := f.Keys[str]; ok {
			delete(keyId, id)
			f.setChanged(true)
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

func (f *KeyField) setChanged(changed bool) {
	f.changed = changed
}

func (f KeyField) Save() error {

	file, err := os.Create(fmt.Sprintf("data/facets/key-%d.jz", f.Id))
	if err != nil {
		return err
	}

	defer file.Close()
	//zipWriter := gzip.NewWriter(file)
	//defer zipWriter.Close()
	return json.NewEncoder(file).Encode(f)
}

func EmptyKeyValueField(field *types.BaseField) KeyField {
	return KeyField{
		BaseField: field,
		Keys:      map[string]types.ItemList{},
	}
}
