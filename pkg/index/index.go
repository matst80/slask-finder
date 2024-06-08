package index

import (
	"log"
	"time"

	"tornberg.me/facet-search/pkg/facet"
)

type Index struct {
	Fields       map[int64]facet.ValueField
	NumberFields map[int64]facet.NumberValueField
	BoolFields   map[int64]facet.BoolValueField
	Items        map[int64]Item
	itemIds      []int64
}

func NewIndex() *Index {
	return &Index{
		Fields:       make(map[int64]facet.ValueField),
		NumberFields: make(map[int64]facet.NumberValueField),
		BoolFields:   make(map[int64]facet.BoolValueField),
		Items:        make(map[int64]Item),
		itemIds:      []int64{},
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
			log.Printf("Field not found %v", key)
		}
	}
	for key, value := range item.NumberFields {
		if f, ok := i.NumberFields[key]; ok {
			f.AddValueLink(value, item.Id)
		} else {
			log.Printf("NumberField not found %v", key)
		}
	}

	for key, value := range item.BoolFields {
		if f, ok := i.BoolFields[key]; ok {
			f.AddValueLink(value, item.Id)
		} else {
			log.Printf("BoolField not found %v", key)
		}
	}
}

func (i *Index) AddItem(item Item) {
	i.Items[item.Id] = item
	i.itemIds = append(i.itemIds, item.Id)
	i.AddItemValues(item)
}

func (i *Index) GetItem(id int64) Item {
	return i.Items[id]
}

func (i *Index) HasItem(id int64) bool {
	return i.Items[id].Id == id
}

func (i *Index) GetItems(ids []int64, page int, pageSize int) []Item {
	items := []Item{}
	l := len(ids)
	start := page * pageSize
	end := min(l, (page+1)*pageSize)
	if start > l {
		return items
	}
	for _, id := range ids[start:end] {
		items = append(items, i.Items[id])
	}
	return items
}

type Facets struct {
	Fields       map[int64]facet.ValueField       `json:"fields"`
	NumberFields map[int64]facet.NumberValueField `json:"numberFields"`
	BoolFields   map[int64]facet.BoolValueField   `json:"boolFields"`
}

func (i *Index) GetFacetsFromResult(result facet.Result) Facets {
	start := time.Now()
	r := Facets{
		Fields:       map[int64]facet.ValueField{},
		NumberFields: map[int64]facet.NumberValueField{},
		BoolFields:   map[int64]facet.BoolValueField{},
	}
	for _, id := range result.Ids() {
		item := i.Items[id] // todo optimize and maybe sort in this step
		for key, value := range item.Fields {
			if f, ok := r.Fields[key]; ok {
				f.AddValueLink(value, id)
			} else {
				r.Fields[key] = facet.NewValueField(i.Fields[key].Field, value, id)
			}
		}
		for key, value := range item.NumberFields {
			if f, ok := r.NumberFields[key]; ok {
				f.AddValueLink(value)
			} else {
				r.NumberFields[key] = facet.NewNumberValueField(i.NumberFields[key].Field, value)
			}
		}
		for key, value := range item.BoolFields {
			if f, ok := r.BoolFields[key]; ok {
				f.AddValueLink(value)
			} else {
				r.BoolFields[key] = facet.NewBoolValueField(i.BoolFields[key].Field, value)
			}
		}
	}
	log.Printf("GetFacetsFromResultIds took %v", time.Since(start))
	return r
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
