package main

import (
	"sort"

	"tornberg.me/facet-search/pkg/facet"
	"tornberg.me/facet-search/pkg/index"
)

func MakeSortMap(items map[uint]*index.DataItem, fieldId uint, fn func(value int) float64) facet.ByValue {
	l := len(items)

	sortMap := make(facet.ByValue, l)
	idx := 0
	for _, item := range items {
		b := 0.0
		if item.SaleStatus == "ACT" {
			b = 5000000.0
		}
		v := 0

		for _, f := range item.IntegerFields {
			if f.Id == fieldId {
				v = f.Value
				break
			}
		}
		sortMap[idx] = facet.Lookup{Id: item.Id, Value: b + fn(v)}
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
	sortMap := MakeSortMap(items, fieldId, func(value int) float64 {
		j += 1.0
		return (float64(value) / 1000.0) + j
	})
	l := len(sortMap)

	sort.Sort(sortMap)
	sortIndex := make(facet.SortIndex, l)
	for idx, item := range sortMap {
		sortIndex[idx] = item.Id
	}
	return sortMap, sortIndex
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
