package index

import (
	"cmp"
	"fmt"
	"slices"

	"tornberg.me/facet-search/pkg/facet"
	"tornberg.me/facet-search/pkg/types"
)

type NumberSearch[K float64 | int] struct {
	facet.NumberRange[K]
	Id uint `json:"id"`
}

type StringSearch struct {
	Id    uint   `json:"id"`
	Value string `json:"value"`
}

type Filters struct {
	StringFilter  []StringSearch          `json:"string" schema:"-"`
	NumberFilter  []NumberSearch[float64] `json:"number" schema:"-"`
	IntegerFilter []NumberSearch[int]     `json:"integer" schema:"-"`
}

func (i *Index) Match(search *Filters, initialIds *types.ItemList, idList chan<- *types.ItemList) {
	cnt := 0
	i.mu.Lock()
	defer i.mu.Unlock()
	results := make(chan *types.ItemList)

	parseKeys := func(field StringSearch, fld types.Facet) {
		results <- fld.Match(field.Value)
	}
	parseInts := func(field NumberSearch[int], fld types.Facet) {
		results <- fld.Match(field.NumberRange)
	}
	parseNumber := func(field NumberSearch[float64], fld types.Facet) {
		results <- fld.Match(field.NumberRange)
	}
	for _, fld := range search.StringFilter {
		if f, ok := i.Facets[fld.Id]; ok {
			cnt++
			go parseKeys(fld, f)
		}
	}
	for _, fld := range search.IntegerFilter {
		if f, ok := i.Facets[fld.Id]; ok {
			cnt++
			go parseInts(fld, f)
		}
	}

	for _, fld := range search.NumberFilter {
		if f, ok := i.Facets[fld.Id]; ok {

			cnt++
			go parseNumber(fld, f)
		}
	}
	if initialIds != nil && len(*initialIds) > 0 {
		if cnt == 0 {
			idList <- initialIds
			return
		}
		cnt++
		go func() {
			results <- initialIds
		}()
	}

	idList <- types.MakeIntersectResult(results, cnt)

}

type KeyFieldWithValue struct {
	types.Facet
	Value interface{}
}

func (i *Index) Related(id uint) (*types.ItemList, error) {
	i.Lock()
	defer i.Unlock()
	item, ok := i.Items[uint(id)]
	if !ok {
		return nil, fmt.Errorf("Item with id %d not found", id)
	}
	fields := make([]KeyFieldWithValue, 0)
	result := types.ItemList{}
	var base *types.BaseField
	for id, itemField := range (*item).GetFields() {
		field, ok := i.Facets[id]
		if !ok {
			continue
		}
		base = field.GetBaseField()
		if ok && base.CategoryLevel != 1 {
			fields = append(fields, KeyFieldWithValue{
				Facet: field,
				Value: itemField,
			})
		}
	}
	slices.SortFunc(fields, func(a, b KeyFieldWithValue) int {
		return cmp.Compare(b.GetBaseField().Priority, a.GetBaseField().Priority)
	})
	if len(fields) == 0 {
		return &result, nil
	}

	first := fields[0]
	result = *first.Match(first.Value)
	for _, field := range fields[1:] {
		if len(result) < 500 {
			return &result, nil
		}
		next := field.Match(field.Value)
		result.Intersect(*next)
	}
	return &result, nil
}
