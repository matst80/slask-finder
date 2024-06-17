package main

import (
	"math"
	"sort"

	"tornberg.me/facet-search/pkg/facet"
	"tornberg.me/facet-search/pkg/index"
)

func MakeSortFromNumberField(items map[uint]*index.DataItem, fieldId uint) facet.SortIndex {
	l := len(items)
	sortIndex := make(facet.SortIndex, l)
	sortMap := make(facet.ByValue, l)
	idx := 0
	for _, item := range items {
		v := 0
		for _, f := range *item.IntegerFields {
			if f.Id == fieldId {
				v = f.Value
				break
			}
		}
		sortMap[idx] = facet.Lookup{Id: item.Id, Value: math.Abs(float64(v) - 300000.0)}
		idx++
	}
	sort.Sort(sortMap[:idx])
	for idx, item := range sortMap {
		sortIndex[idx] = item.Id
	}
	return sortIndex
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
