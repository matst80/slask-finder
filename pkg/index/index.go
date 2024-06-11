package index

import (
	"hash/fnv"
	"log"
	"sort"
	"time"

	"tornberg.me/facet-search/pkg/facet"
)

type Index struct {
	KeyFacets     map[int]*facet.KeyField
	DecimalFacets map[int]*facet.NumberField[float64]
	IntFacets     map[int]*facet.NumberField[int]

	Items map[int]Item
}

func NewIndex() *Index {
	return &Index{
		KeyFacets:     make(map[int]*facet.KeyField),
		DecimalFacets: make(map[int]*facet.NumberField[float64]),
		IntFacets:     make(map[int]*facet.NumberField[int]),

		Items: make(map[int]Item),
	}
}

func (i *Index) AddKeyField(field *facet.BaseField) {
	i.KeyFacets[field.Id] = facet.EmptyKeyValueField(field)
}

func (i *Index) AddDecimalField(field *facet.BaseField) {
	i.DecimalFacets[field.Id] = facet.EmptyNumberField[float64](field)
}

func (i *Index) AddIntegerField(field *facet.BaseField) {
	i.IntFacets[field.Id] = facet.EmptyNumberField[int](field)
}

func (i *Index) AddItemValues(item DataItem) {

	for key, value := range item.Fields {
		if f, ok := i.KeyFacets[key]; ok {
			f.AddValueLink(value, item.Id)
		} else {
			//delete(item.Fields, key)
			//log.Printf("Field not found %v: %v", key, value)
		}
	}
	for key, value := range item.DecimalFields {
		if f, ok := i.DecimalFacets[key]; ok {
			f.AddValueLink(value, item.Id)
		} else {
			//log.Printf("DecimalField not found %v: %v", key, value)
			//delete(item.NumberFields, key)
		}
	}

	for key, value := range item.IntegerFields {
		if f, ok := i.IntFacets[key]; ok {
			f.AddValueLink(value, item.Id)
		} else {
			//log.Printf("IntField not found %v: %v", key, value)
			//delete(item.NumberFields, key)
		}
	}

}

func HashString(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func getFields(fields map[int]*facet.KeyField, itemFields map[int]string) map[int]ItemKeyField {
	newFields := make(map[int]ItemKeyField)
	for key, value := range itemFields {
		if f, ok := fields[key]; ok {
			newFields[key] = ItemKeyField{field: f, Value: value, ValueHash: HashString(value)}
		}
	}
	return newFields
}

func getNumberFields[K facet.FieldNumberValue](fields map[int]*facet.NumberField[K], itemFields map[int]K) map[int]ItemNumberField[K] {
	newFields := make(map[int]ItemNumberField[K])
	for key, value := range itemFields {
		if f, ok := fields[key]; ok {
			newFields[key] = ItemNumberField[K]{field: f, Value: value}
		}
	}
	return newFields
}

func (i *Index) AddItem(item DataItem) {

	i.AddItemValues(item)
	i.Items[item.Id] = Item{
		BaseItem:      item.BaseItem,
		Fields:        getFields(i.KeyFacets, item.Fields),
		DecimalFields: getNumberFields(i.DecimalFacets, item.DecimalFields),
		IntegerFields: getNumberFields(i.IntFacets, item.IntegerFields),
	}
}

func (i *Index) GetItem(id int) Item {
	return i.Items[id]
}

func (i *Index) HasItem(id int) bool {
	return i.Items[id].Id == id
}

func (i *Index) GetItemIds(ids []int, page int, pageSize int) []int {
	l := len(ids)
	start := page * pageSize
	end := min(l, start+pageSize)
	if start > l {
		return ids[0:0]
	}
	return ids[start:end]
}

func getFieldValues(item *Item) map[int]interface{} {
	fields := map[int]interface{}{}
	for key, value := range item.Fields {
		fields[key] = value.Value
	}
	for key, value := range item.DecimalFields {
		fields[key] = value.Value
	}
	for key, value := range item.IntegerFields {
		fields[key] = value.Value
	}

	return fields
}

func (i *Index) GetItems(ids []int, page int, pageSize int) []ResultItem {
	items := make([]ResultItem, min(len(ids), pageSize))
	idx := 0
	for _, id := range i.GetItemIds(ids, page, pageSize) {
		item, ok := i.Items[id]
		if ok {
			items[idx] = ResultItem{
				BaseItem: item.BaseItem,
				Fields:   getFieldValues(&item),
			}
			idx++
		}
	}
	return items[0:idx]
}

type hashKeyResult struct {
	value string
	count int
}
type KeyResult struct {
	*facet.BaseField
	values map[uint32]*hashKeyResult
}

func (k *KeyResult) GetValues() map[string]int {
	values := make(map[string]int)
	for _, v := range k.values {
		values[v.value] = v.count
	}
	return values
}

type JsonKeyResult struct {
	*facet.BaseField
	Values map[string]int `json:"values"`
}

func (k *KeyResult) AddValue(hash uint32, value string) {
	if v, ok := k.values[hash]; ok {
		v.count++
	} else {
		k.values[hash] = &hashKeyResult{value: value, count: 1}
	}
	//k.Values[value]++
}

type NumberResult[V float64 | int] struct {
	*facet.BaseField
	Count int `json:"count"`
	Min   V   `json:"min"`
	Max   V   `json:"max"`
}

func (k *NumberResult[V]) AddValue(value V) {
	if value < k.Min {
		k.Min = value
	} else if value > k.Max {
		k.Max = value
	}
	k.Count++
}

type Facets struct {
	Fields       []JsonKeyResult         `json:"fields"`
	NumberFields []NumberResult[float64] `json:"numberFields"`
	IntFields    []NumberResult[int]     `json:"integerFields"`
}

func (i *Index) GetFacetsFromResult(ids *facet.IdList, filters *Filters, sortIndex *facet.SortIndex) Facets {
	start := time.Now()
	count := 0
	fields := map[int]KeyResult{}
	numberFields := map[int]NumberResult[float64]{}
	intFields := map[int]NumberResult[int]{}

	for id := range *ids {

		item, ok := i.Items[id] // todo optimize and maybe sort in this step
		if !ok {
			continue
		}

		for fieldId, field := range item.Fields {
			if field.Value == "" {
				continue
			}
			if f, ok := fields[fieldId]; ok {
				f.AddValue(field.ValueHash, field.Value)
			} else {
				count++
				fields[fieldId] = KeyResult{
					BaseField: field.field.BaseField,
					values:    map[uint32]*hashKeyResult{field.ValueHash: {value: field.Value, count: 1}},
				}
			}
		}

		for key, field := range item.DecimalFields {
			if f, ok := numberFields[key]; ok {
				f.AddValue(field.Value)
			} else {
				count++
				numberFields[key] = NumberResult[float64]{
					BaseField: field.field.BaseField,
					Count:     1,
					Min:       field.Value,
					Max:       field.Value,
				}
			}
		}
		for key, field := range item.IntegerFields {
			if f, ok := intFields[key]; ok {
				f.AddValue(field.Value)
			} else {
				count++
				intFields[key] = NumberResult[int]{
					BaseField: field.field.BaseField,
					Count:     1,
					Min:       field.Value,
					Max:       field.Value,
				}
			}
		}

	}
	log.Printf("GetFacetsFromResultIds took %v, found %d facets", time.Since(start), len(fields)+len(numberFields)+len(intFields))

	return Facets{
		Fields:       mapToSlice(fields, sortIndex),
		NumberFields: mapToSliceNumber(numberFields, sortIndex),
		IntFields:    mapToSliceNumber(intFields, sortIndex),
	}
}

type NumberSearch[K float64 | int] struct {
	Id  int `json:"id"`
	Min K   `json:"min"`
	Max K   `json:"max"`
}

type StringSearch struct {
	Id    int    `json:"id"`
	Value string `json:"value"`
}

type BoolSearch struct {
	Id    int  `json:"id"`
	Value bool `json:"value"`
}

type Filters struct {
	StringFilter  []StringSearch          `json:"string"`
	NumberFilter  []NumberSearch[float64] `json:"number"`
	IntegerFilter []NumberSearch[int]     `json:"integer"`
	BoolFilter    []BoolSearch            `json:"bool"`
}

func (i *Index) MakeSortForFields() facet.SortIndex {

	l := len(i.DecimalFacets) + len(i.KeyFacets) + len(i.IntFacets)
	idx := 0
	sortIndex := make(facet.SortIndex, l)
	sortMap := make(facet.ByValue, l)

	for _, item := range i.DecimalFacets {
		sortMap[idx] = facet.Lookup{Id: item.Id, Value: float64(item.TotalCount())}
		idx++
	}
	for _, item := range i.KeyFacets {
		sortMap[idx] = facet.Lookup{Id: item.Id, Value: float64(item.TotalCount())}
		idx++
	}
	for _, item := range i.IntFacets {
		sortMap[idx] = facet.Lookup{Id: item.Id, Value: float64(item.TotalCount())}
		idx++
	}
	sort.Sort(sort.Reverse(sortMap))
	for idx, item := range sortMap {
		sortIndex[idx] = item.Id
	}
	return sortIndex
}

type IndexMatch struct {
	Ids facet.IdList
}

func (i *Index) Match(search *Filters) facet.IdList {
	len := 0
	results := make(chan facet.IdList)

	parseKeys := func(field StringSearch, fld *facet.KeyField) {
		start := time.Now()
		results <- fld.Matches(field.Value)
		log.Printf("String match took %v", time.Since(start))
	}
	parseInts := func(field NumberSearch[int], fld *facet.NumberField[int]) {
		start := time.Now()
		results <- fld.MatchesRange(field.Min, field.Max)
		log.Printf("Integer match took %v", time.Since(start))
	}
	parseNumber := func(field NumberSearch[float64], fld *facet.NumberField[float64]) {
		start := time.Now()
		results <- fld.MatchesRange(field.Min, field.Max)
		log.Printf("Decimal match took %v", time.Since(start))
	}
	for _, fld := range search.StringFilter {
		if f, ok := i.KeyFacets[fld.Id]; ok {
			len++
			go parseKeys(fld, f)
		}
	}
	for _, fld := range search.IntegerFilter {
		if f, ok := i.IntFacets[fld.Id]; ok {
			len++
			go parseInts(fld, f)
		}
	}

	for _, fld := range search.NumberFilter {
		if f, ok := i.DecimalFacets[fld.Id]; ok {
			len++
			go parseNumber(fld, f)
		}
	}

	return facet.MakeIntersectResult(results, len)

}
