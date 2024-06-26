package index

import (
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

type BoolSearch struct {
	Id    uint `json:"id"`
	Value bool `json:"value"`
}

type Filters struct {
	StringFilter  []StringSearch          `json:"string"`
	NumberFilter  []NumberSearch[float64] `json:"number"`
	IntegerFilter []NumberSearch[int]     `json:"integer"`
}

func (i *Index) Match(search *Filters, initialIds *facet.IdList) *facet.IdList {
	cnt := 0
	i.mu.Lock()
	defer i.mu.Unlock()
	results := make(chan *facet.IdList)
	if initialIds != nil && len(*initialIds) > 0 {

		cnt++
		go func() {
			results <- initialIds
		}()
	}
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
			cnt++
			go parseInts(fld, f)
		}
	}

	for _, fld := range search.NumberFilter {
		if f, ok := i.DecimalFacets[fld.Id]; ok {
			cnt++
			go parseNumber(fld, f)
		}
	}
	if cnt == 0 {
		list := facet.IdList{}
		for id := range i.AllItems {
			list.Add(id)
		}
		return &list
	}
	return facet.MakeIntersectResult(results, cnt)

}
