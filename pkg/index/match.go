package index

import (
	"log"
	"time"

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
	BoolFilter    []BoolSearch            `json:"bool"`
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
