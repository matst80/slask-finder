package index

import (
	"log"

	"tornberg.me/facet-search/pkg/facet"
)

type Index struct {
	Fields       map[int64]facet.ValueField
	NumberFields map[int64]facet.NumberValueField
	Items        map[int64]Item
	itemIds      []int64
}

func NewIndex() *Index {
	return &Index{
		Fields:       make(map[int64]facet.ValueField),
		NumberFields: make(map[int64]facet.NumberValueField),
		Items:        make(map[int64]Item),
		itemIds:      []int64{},
	}
}

func (i *Index) AddField(id int64, field facet.Field) {
	i.Fields[id] = facet.EmptyValueField(field)
}

func (i *Index) AddNumberField(id int64, field facet.Field) {
	i.NumberFields[id] = facet.EmptyNumberField(field)
}

func (i *Index) AddItem(item Item) {

	i.Items[item.Id] = item
	i.itemIds = append(i.itemIds, item.Id)
	for _, field := range item.Fields {

		if f, ok := i.Fields[field.Id]; ok {
			f.AddValueLink(field.Value, item.Id)
		} else {
			log.Fatalf("Field not found %v", field.Id)
		}
	}
	for _, field := range item.NumberFields {
		if f, ok := i.NumberFields[field.Id]; ok {
			f.AddValueLink(field.Value, item.Id)
		} else {
			log.Fatalf("NumberField not found %v", field.Id)
			//i.NumberFields[field.Id] = facet.NewNumberValueField(facet.Field{}, field.Value, item.Id)
		}
	}
}

func (i *Index) GetItem(id int64) Item {
	return i.Items[id]
}

func (i *Index) HasItem(id int64) bool {
	return i.Items[id].Id == id
}

func (i *Index) GetItems(ids []int64) []Item {
	items := []Item{}
	for _, id := range ids {
		items = append(items, i.Items[id])
	}
	return items
}

type Facets struct {
	Fields       map[int64]facet.ValueField       `json:"fields"`
	NumberFields map[int64]facet.NumberValueField `json:"numberFields"`
}

func (i *Index) GetFacetsFromResultIds(ids []int64) Facets {
	r := Facets{
		Fields:       map[int64]facet.ValueField{},
		NumberFields: map[int64]facet.NumberValueField{},
	}
	for _, id := range ids {
		item := i.Items[id]
		for _, field := range item.Fields {
			if f, ok := r.Fields[field.Id]; ok {
				f.AddValueLink(field.Value, item.Id)
			} else {
				r.Fields[field.Id] = facet.NewValueField(i.Fields[field.Id].Field, field.Value, item.Id)
			}
		}
		for _, numberField := range item.NumberFields {
			if f, ok := r.NumberFields[numberField.Id]; ok {
				f.AddValueLink(numberField.Value, item.Id)
			} else {
				r.NumberFields[numberField.Id] = facet.NewNumberValueField(i.NumberFields[numberField.Id].Field, numberField.Value, item.Id)
			}
		}
	}
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

func (i *Index) Match(strings []StringSearch, numbers []NumberSearch) []int64 {
	result := facet.Result{Ids: i.itemIds}
	for _, field := range numbers {
		if f, ok := i.NumberFields[field.Id]; ok {
			if len(result.Ids) > 0 {
				result.Intersect(f.Matches(field.Min, field.Max))
			}
		}
	}
	for _, field := range strings {
		if f, ok := i.Fields[field.Id]; ok {
			if len(result.Ids) > 0 {
				result.Intersect(f.Matches(field.Value))
			}
		}
	}

	return result.Ids
}
