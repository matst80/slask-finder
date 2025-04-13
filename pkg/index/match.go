package index

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/matst80/slask-finder/pkg/types"
)

type CleanKeyFacet struct {
	types.Facet
	level int
	Value interface{} `json:"value"`
}

func (i *Index) RemoveDuplicateCategoryFilters(stringFilters []types.StringFilter) []CleanKeyFacet {
	ret := make([]CleanKeyFacet, 0, len(stringFilters))

	maxLevel := 0
	for _, fld := range stringFilters {
		if fld.Value == nil {
			continue
		}
		if f, ok := i.Facets[fld.Id]; ok && f != nil {
			level := f.GetBaseField().CategoryLevel
			ret = append(ret, CleanKeyFacet{
				Facet: f,
				Value: fld.Value,
				level: level,
			})

			if level > 0 {
				if level > maxLevel {
					maxLevel = level
				}
			}
		}
	}

	return slices.DeleteFunc(ret, func(f CleanKeyFacet) bool {
		return f.level > 0 && f.level < maxLevel
	})
}

func (i *Index) Match(search *types.Filters, initialIds *types.ItemList, idList chan<- *types.ItemList) {
	cnt := 0
	i.mu.Lock()
	defer i.mu.Unlock()
	results := make(chan *types.ItemList)

	parseKeys := func(value interface{}, facet types.Facet) {
		results <- facet.Match(value)
	}
	parseRange := func(field types.RangeFilter, facet types.Facet) {
		results <- facet.Match(field)
	}

	for _, fld := range i.RemoveDuplicateCategoryFilters(search.StringFilter) {
		cnt++
		go parseKeys(fld.Value, fld.Facet)
	}

	for _, fld := range search.RangeFilter {
		if f, ok := i.Facets[fld.Id]; ok && f != nil {
			cnt++
			go parseRange(fld, f)
		}
	}
	if initialIds != nil {
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
	item, ok := i.Items[id]
	if !ok {
		return nil, fmt.Errorf("Item with id %d not found", id)
	}
	fields := make([]KeyFieldWithValue, 0)
	result := types.ItemList{}
	var base *types.BaseField
	for id, itemField := range (*item).GetFields() {
		field, ok := i.Facets[id]
		if !ok || field.GetType() != types.FacetKeyType {
			continue
		}
		base = field.GetBaseField()
		if base.CategoryLevel != 1 {
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

func (i *Index) Compatible(id uint) (*types.ItemList, error) {
	i.Lock()
	defer i.Unlock()
	item, ok := i.Items[id]
	if !ok {
		return nil, fmt.Errorf("Item with id %d not found", id)
	}
	fields := make([]KeyFieldWithValue, 0)
	result := types.ItemList{}
	var base *types.BaseField
	for id, itemField := range (*item).GetFields() {
		field, ok := i.Facets[id]
		if !ok || field.GetType() != types.FacetKeyType {
			continue
		}
		base = field.GetBaseField()
		if base.LinkedId > 0 {
			fields = append(fields, KeyFieldWithValue{
				Facet: field,
				Value: itemField,
			})
		}
	}
	// slices.SortFunc(fields, func(a, b KeyFieldWithValue) int {
	// 	return cmp.Compare(b.GetBaseField().Priority, a.GetBaseField().Priority)
	// })
	if len(fields) == 0 {
		return &result, nil
	}

	first := fields[0]
	result = *first.Match(first.Value)
	for _, field := range fields[1:] {
		// if len(result) < 500 {
		// 	return &result, nil
		// }
		next := field.Match(field.Value)
		if next != nil && result.HasIntersection(next) {
			result.Intersect(*next)
		}
	}
	return &result, nil
}
