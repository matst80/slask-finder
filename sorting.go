package main

import (
	"sort"

	"tornberg.me/facet-search/pkg/facet"
	"tornberg.me/facet-search/pkg/index"
)

func MakeSortMap(items map[uint]*index.DataItem, fieldId uint, fn func(value int, item *index.DataItem) float64) facet.ByValue {
	l := len(items)

	sortMap := make(facet.ByValue, l)
	idx := 0
	for _, item := range items {

		v := 0

		for _, f := range item.IntegerFields {
			if f.Id == fieldId {
				v = f.Value
				break
			}
		}
		sortMap[idx] = facet.Lookup{Id: item.Id, Value: fn(v, item)}
		idx++
	}
	return sortMap[:idx]
}

func ToMap(f *facet.ByValue) map[uint]float64 {
	m := make(map[uint]float64)
	for _, item := range *f {
		m[item.Id] = item.Value
	}
	return m
}

func MakeSortFromNumberField(items map[uint]*index.DataItem, fieldId uint) (facet.ByValue, facet.SortIndex) {
	j := 0.0
	sortMap := MakeSortMap(items, fieldId, func(value int, item *index.DataItem) float64 {
		b := 0.0
		if item.SaleStatus == "ACT" {
			b = 50000.0
		}
		j += 0.0001
		return b + float64(value) + j
	})
	l := len(sortMap)

	sort.Sort(sortMap)
	sortIndex := make(facet.SortIndex, l)
	for idx, item := range sortMap {
		sortIndex[idx] = item.Id
	}
	return sortMap, sortIndex
}

func ToSortIndex(f *facet.ByValue, reversed bool) *facet.SortIndex {
	l := len(*f)
	if reversed {
		sort.Sort(sort.Reverse(*f))
	} else {
		sort.Sort(*f)
	}

	sortIndex := make(facet.SortIndex, l)
	for idx, item := range *f {
		sortIndex[idx] = item.Id
	}
	return &sortIndex
}

func MakeSortMaps(items map[uint]*index.DataItem) map[string]*facet.SortIndex {
	j := 0.00000
	sortMap := MakeSortMap(items, 4, func(value int, item *index.DataItem) float64 {
		j += 0.0001
		return float64(value) + j
	})
	ret := make(map[string]*facet.SortIndex)
	ret["price"] = ToSortIndex(&sortMap, true)
	ret["price_desc"] = ToSortIndex(&sortMap, false)
	return ret
}

func MakeSortForFields() facet.SortIndex {

	l := len(idx.DecimalFacets) + len(idx.KeyFacets) + len(idx.IntFacets)
	i := 0
	sortIndex := make(facet.SortIndex, l)
	sortMap := make(facet.ByValue, l)

	for _, item := range idx.DecimalFacets {
		sortMap[i] = facet.Lookup{Id: item.Id, Value: float64(item.TotalCount())}
		i++
	}
	for _, item := range idx.KeyFacets {
		sortMap[i] = facet.Lookup{Id: item.Id, Value: float64(item.TotalCount())}
		i++
	}
	for _, item := range idx.IntFacets {
		sortMap[i] = facet.Lookup{Id: item.Id, Value: float64(item.TotalCount())}
		i++
	}
	sort.Sort(sort.Reverse(sortMap))
	for idx, item := range sortMap {
		sortIndex[idx] = item.Id
	}
	return sortIndex
}
