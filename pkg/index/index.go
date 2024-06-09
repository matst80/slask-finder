package index

import (
	"log"
	"sort"
	"time"

	"tornberg.me/facet-search/pkg/facet"
)

type Index struct {
	Fields       map[int64]facet.ValueField
	NumberFields map[int64]facet.NumberValueField
	BoolFields   map[int64]facet.BoolValueField
	Items        map[int64]Item
}

func NewIndex() *Index {
	return &Index{
		Fields:       make(map[int64]facet.ValueField),
		NumberFields: make(map[int64]facet.NumberValueField),
		BoolFields:   make(map[int64]facet.BoolValueField),
		Items:        make(map[int64]Item),
	}
}

func (i *Index) AddField(field facet.Field) {
	i.Fields[field.Id] = facet.EmptyValueField(field)
}

func (i *Index) AddBoolField(field facet.Field) {
	i.BoolFields[field.Id] = facet.EmptyBoolField(field)
}

func (i *Index) AddNumberField(field facet.Field) {
	i.NumberFields[field.Id] = facet.EmptyNumberField(field)
}

func (i *Index) AddItemValues(item Item) {

	for key, value := range item.Fields {
		if f, ok := i.Fields[key]; ok {
			f.AddValueLink(value, item.Id)
		} else {
			delete(item.Fields, key)
			//log.Printf("Field not found %v", key)
		}
	}
	for key, value := range item.NumberFields {
		if f, ok := i.NumberFields[key]; ok {
			f.AddValueLink(value, item.Id)
		} else {
			//log.Printf("NumberField not found %v", key)
			delete(item.NumberFields, key)
		}
	}

	for key, value := range item.BoolFields {
		if f, ok := i.BoolFields[key]; ok {
			f.AddValueLink(value, item.Id)
		} else {
			//log.Printf("BoolField not found %v", key)
			delete(item.BoolFields, key)
		}
	}
}

func (i *Index) AddItem(item Item) {
	i.Items[item.Id] = item
	i.AddItemValues(item)
}

func (i *Index) GetItem(id int64) Item {
	return i.Items[id]
}

func (i *Index) HasItem(id int64) bool {
	return i.Items[id].Id == id
}

func (i *Index) GetItemIds(ids []int64, page int, pageSize int) []int64 {
	l := len(ids)
	start := page * pageSize
	end := min(l, start+pageSize)
	if start > l {
		return ids[0:0]
	}
	return ids[start:end]
}

func (i *Index) GetItems(ids []int64, page int, pageSize int) []Item {
	items := []Item{}

	for _, id := range i.GetItemIds(ids, page, pageSize) {
		items = append(items, i.Items[id])
	}
	return items
}

type StringResult struct {
	Field  facet.Field    `json:"field"`
	Values map[string]int `json:"values"`
}

type NumberResult struct {
	Field facet.Field `json:"field"`
	Count int         `json:"count"`
	Min   float64     `json:"min"`
	Max   float64     `json:"max"`
}

type BoolResult struct {
	Field  facet.Field    `json:"field"`
	Values map[string]int `json:"values"`
}

type Facets struct {
	Fields       []StringResult  `json:"fields"`
	NumberFields []*NumberResult `json:"numberFields"`
	BoolFields   []BoolResult    `json:"boolFields"`
}

func (i *Index) GetFacetsFromResult(result facet.Result, sortIndex facet.SortIndex) Facets {
	start := time.Now()
	all := result.Ids()
	ids := all[0:min(1000, len(all))]

	fields := map[int64]StringResult{}
	numberFields := map[int64]*NumberResult{}
	boolFields := map[int64]BoolResult{}

	for _, id := range ids {
		item, ok := i.Items[id] // todo optimize and maybe sort in this step
		if !ok {
			continue
		}
		for key, value := range item.Fields {
			if f, ok := fields[key]; ok {
				f.Values[value]++
			} else {
				fields[key] = StringResult{
					Field:  i.Fields[key].Field,
					Values: map[string]int{value: 1},
				}
			}
		}
		for key, value := range item.NumberFields {
			if f, ok := numberFields[key]; ok {
				if value < f.Min {
					f.Min = value
				} else if value > f.Max {
					f.Max = value
				}
				f.Count++
			} else {
				numberFields[key] = &NumberResult{
					Field: i.NumberFields[key].Field,
					Count: 1,
					Min:   value,
					Max:   value,
				}
			}
		}
		for key, value := range item.BoolFields {
			if f, ok := boolFields[key]; ok {
				f.Values[stringValue(value)]++
			} else {
				boolFields[key] = BoolResult{
					Field:  i.BoolFields[key].Field,
					Values: map[string]int{stringValue(value): 1},
				}
			}
		}
	}
	log.Printf("GetFacetsFromResultIds took %v", time.Since(start))

	return Facets{
		Fields:       mapToSlice(fields, sortIndex),
		NumberFields: mapToSliceRef(numberFields, sortIndex),
		BoolFields:   mapToSlice(boolFields, sortIndex),
	}
}

func stringValue(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func mapToSliceRef[V BoolResult | StringResult | NumberResult](fields map[int64]*V, sortIndex facet.SortIndex) []*V {

	l := min(len(fields), 256)
	sorted := make([]*V, len(fields))

	idx := 0

	for _, id := range sortIndex {
		if idx >= l {
			break
		}
		f, ok := fields[id]
		if ok {
			sorted[idx] = f

			idx++

		}
	}
	return sorted

}

func mapToSlice[V BoolResult | StringResult | NumberResult](fields map[int64]V, sortIndex facet.SortIndex) []V {

	l := min(len(fields), 256)
	sorted := make([]V, len(fields))

	idx := 0

	for _, id := range sortIndex {
		if idx >= l {
			break
		}
		f, ok := fields[id]
		if ok {
			sorted[idx] = f

			idx++

		}
	}
	return sorted
}

type NumberSearch struct {
	Id  int64   `json:"id"`
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

type StringSearch struct {
	Id    int64  `json:"id"`
	Value string `json:"value"`
}

type BoolSearch struct {
	Id    int64 `json:"id"`
	Value bool  `json:"value"`
}

func (i *Index) MakeSortForFields() facet.SortIndex {

	l := len(i.BoolFields) + len(i.NumberFields) + len(i.Fields)
	idx := 0
	sortIndex := make(facet.SortIndex, l)
	sortMap := make(facet.ByValue, l)
	for _, item := range i.BoolFields {
		sortMap[idx] = facet.Lookup{Id: item.Id, Value: float64(item.TotalCount())}
		idx++
	}
	for _, item := range i.NumberFields {
		sortMap[idx] = facet.Lookup{Id: item.Id, Value: float64(item.TotalCount())}
		idx++
	}
	for _, item := range i.Fields {
		sortMap[idx] = facet.Lookup{Id: item.Id, Value: float64(item.TotalCount())}
		idx++
	}
	sort.Sort(sortMap)
	for idx, item := range sortMap {
		sortIndex[idx] = item.Id
	}
	return sortIndex
}

func (i *Index) Match(strings []StringSearch, numbers []NumberSearch, bits []BoolSearch) facet.Result {

	// log.Printf("Items: %v", len(i.itemIds))

	results := make(chan facet.Result)
	len := 0
	for _, fld := range bits {
		if f, ok := i.BoolFields[fld.Id]; ok {
			len++
			go func(field BoolSearch) {

				start := time.Now()
				results <- f.Matches(field.Value)

				log.Printf("Match took %v", time.Since(start))

			}(fld)
			//}
		}
	}
	for _, fld := range numbers {
		if f, ok := i.NumberFields[fld.Id]; ok {
			len++
			go func(field NumberSearch) {

				start := time.Now()
				results <- f.Matches(field.Min, field.Max)

				log.Printf("Match took %v", time.Since(start))

			}(fld)
			//}
		}
	}
	for _, fld := range strings {
		if f, ok := i.Fields[fld.Id]; ok {
			len++
			go func(field StringSearch) {
				start := time.Now()
				results <- f.Matches(field.Value)

				log.Printf("Match took %v", time.Since(start))

			}(fld)
		}
	}
	return facet.MakeIntersectResult(results, len)

}
