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
}

type ItemList map[uint]*Item

func (i *ItemList) Add(item Item) {
	(*i)[item.GetId()] = &item
}

type KeyField struct {
	*BaseField
	keys map[string]*ItemList
	//values []IdList
	//len    uint
}

func (f KeyField) GetType() uint {
	return FacetKeyType
}

func (f KeyField) GetValues() interface{} {
	ret := make([]string, len(f.keys))
	idx := 0
	for value := range f.keys {
		ret[idx] = value
		idx++
	}
	return ret
}

func (f KeyField) Match(input interface{}) *ItemList {
	value, ok := input.(string)
	if ok {

		list, found := f.keys[value]
		if found {
			return list
		}
	}
	return &ItemList{}
}

func (f KeyField) GetBaseField() *BaseField {
	return f.BaseField
}

func (f KeyField) AddValueLink(data interface{}, item Item) bool {
	value, ok := data.(string)
	if !ok {
		return false
	}
	list, found := f.keys[value]
	if !found {
		f.keys[value] = &ItemList{item.GetId(): &item}
	} else {
		list.Add(item)
	}
	return true
}

func (f KeyField) RemoveValueLink(data interface{}, id uint) {
	value, ok := data.(string)
	if !ok {
		return
	}
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

// func (f *KeyField) GetValues() interface{} {
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

func EmptyKeyValueField(field *BaseField) KeyField {
	return KeyField{
		BaseField: field,
		keys:      map[string]*ItemList{},
	}
}
