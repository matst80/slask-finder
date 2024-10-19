package facet

import "unsafe"

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

type LocationStock []struct {
	Id    string `json:"id"`
	Level string `json:"level"`
}

type BaseItem struct {
	Id    uint
	Sku   string
	Title string
	Price int
	Img   string
}

type Item interface {
	GetId() uint
	GetStock() LocationStock
	GetFields() map[uint]interface{}
	IsDeleted() bool
	GetPrice() int
	GetLastUpdated() int64
	GetCreated() int64
	GetPopularity() float64
	GetTitle() string
	ToString() string
	GetBaseItem() BaseItem
	GetItem() interface{}
}

const FacetKeyType = 1
const FacetNumberType = 2
const FacetIntegerType = 3

type Facet interface {
	GetType() uint
	Match(data interface{}) *ItemList
	GetBaseField() *BaseField
	AddValueLink(value interface{}, item Item) bool
	RemoveValueLink(value interface{}, id uint)
	GetValues() interface{}
	Size() int
}

type ItemList map[uint]*Item

func (i *ItemList) Add(item Item) {
	(*i)[item.GetId()] = &item
}

type KeyField struct {
	*BaseField
	keys map[interface{}]ItemList
}

func (f KeyField) GetType() uint {
	return FacetKeyType
}

func (f KeyField) Size() int {
	sum := 0
	for key, ids := range f.keys {
		sum += int(unsafe.Sizeof(ids)) + len(key.(string))
	}
	return sum
}

func (f KeyField) GetValues() interface{} {
	ret := make([]interface{}, len(f.keys))
	idx := 0
	for value := range f.keys {
		ret[idx] = value
		idx++
	}
	return ret
}

func (f KeyField) Match(input interface{}) *ItemList {
	//value, ok := input.(string)
	//if ok {

	list, found := f.keys[input]
	if found {
		return &list
	}
	//}
	return &ItemList{}
}

func (f KeyField) GetBaseField() *BaseField {
	return f.BaseField
}

func (f KeyField) AddValueLink(data interface{}, item Item) bool {
	str, ok := data.(string)
	if !ok {
		return false
	}
	if len(str) > 64 {
		data = str[:61] + "..."
	}
	list, found := f.keys[data]
	if !found {
		f.keys[data] = ItemList{item.GetId(): &item}
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

func EmptyKeyValueField(field *BaseField) KeyField {
	return KeyField{
		BaseField: field,
		keys:      map[interface{}]ItemList{},
	}
}
