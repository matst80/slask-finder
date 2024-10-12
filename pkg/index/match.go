package index

import (
	"cmp"
	"fmt"
	"slices"

	"tornberg.me/facet-search/pkg/facet"
)

type NumberSearch[K float64 | int] struct {
	Id  uint `json:"id"`
	Min K    `json:"min"`
	Max K    `json:"max"`
}

type StringSearch struct {
	Id    uint   `json:"id"`
	Value string `json:"value"`
}

type Filters struct {
	StringFilter  []StringSearch          `json:"string" schema:"str"`
	NumberFilter  []NumberSearch[float64] `json:"number" schema:"num"`
	IntegerFilter []NumberSearch[int]     `json:"integer" schema:"int"`
}

func (i *Index) Match(search *Filters, initialIds *facet.IdList, idList chan<- *facet.IdList) {
	cnt := 0
	i.mu.Lock()
	defer i.mu.Unlock()
	results := make(chan *facet.IdList)

	parseKeys := func(field StringSearch, fld *facet.KeyField) {
		results <- fld.Matches(field.Value)
	}
	parseInts := func(field NumberSearch[int], fld *facet.NumberField[int]) {
		results <- fld.MatchesRange(field.Min, field.Max)
	}
	parseNumber := func(field NumberSearch[float64], fld *facet.NumberField[float64]) {
		results <- fld.MatchesRange(field.Min, field.Max)
	}
	for _, fld := range search.StringFilter {
		if f, ok := i.KeyFacets[fld.Id]; ok {
			cnt++
			go parseKeys(fld, f)
		}
	}
	for _, fld := range search.IntegerFilter {
		if f, ok := i.IntFacets[fld.Id]; ok {
			if (f.Max == fld.Max && f.Min == fld.Min) || (f.Max == 0 && f.Min == 0) || (fld.Min == fld.Max) {
				continue
			}
			cnt++
			go parseInts(fld, f)
		}
	}

	for _, fld := range search.NumberFilter {
		if f, ok := i.DecimalFacets[fld.Id]; ok {
			if (f.Max == fld.Max && f.Min == fld.Min) || (f.Max == 0 && f.Min == 0) || (fld.Min == fld.Max) {
				continue
			}
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

	idList <- facet.MakeIntersectResult(results, cnt)

}

type KeyFieldWithValue struct {
	*facet.KeyField
	Value string
}

func (i *Index) Related(id uint) (*facet.IdList, error) {
	i.Lock()
	defer i.Unlock()
	item, ok := i.Items[uint(id)]
	if !ok {
		return nil, fmt.Errorf("Item with id %d not found", id)
	}
	fields := make([]KeyFieldWithValue, 0)
	result := facet.IdList{}

	for _, itemField := range item.Fields {
		field, ok := i.KeyFacets[itemField.Id]
		if ok && field.CategoryLevel != 1 {
			fields = append(fields, KeyFieldWithValue{
				KeyField: field,
				Value:    itemField.Value,
			})
		}
	}
	slices.SortFunc(fields, func(a, b KeyFieldWithValue) int {
		return cmp.Compare(b.Priority, a.Priority)
	})
	if len(fields) == 0 {
		return &result, nil
	}

	first := fields[0]
	result = *first.Matches(first.Value)
	for _, field := range fields[1:] {
		if len(result) < 500 {
			return &result, nil
		}
		next := field.Matches(field.Value)
		result.Intersect(*next)
	}
	return &result, nil
}
