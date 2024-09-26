package index

import (
	"sort"

	"tornberg.me/facet-search/pkg/facet"
)

func MakeSortMap(items map[uint]*DataItem, fieldId uint, fn func(value int, item *DataItem) float64) facet.ByValue {
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

func MakePopularSortMap(items map[uint]*DataItem) facet.ByValue {
	l := len(items)
	j := 0.0
	sortMap := make(facet.ByValue, l)
	idx := 0
	for _, item := range items {
		j += 0.0000000000001
		v := 0
		price := 0
		//sinceUpdate := 0
		orgPrice := 0
		discount := 1
		grade := 0
		noGrades := 0
		//chanedTs :=  item.LastUpdate
		for _, f := range item.IntegerFields {
			if f.Id == 4 {
				price = f.Value
				break
			}
			if f.Id == 5 {
				orgPrice = f.Value
				break
			}
			if f.Id == 6 {
				grade = f.Value
				break
			}
			if f.Id == 7 {
				noGrades = f.Value
				break
			}
		}
		if orgPrice > 0 {
			discount = orgPrice - price
		}
		if item.SaleStatus == "ACT" {
			v += 5000
		}
		f := float64((discount * 100000) + ((grade - 1) * noGrades * 100) + v)
		sortMap[idx] = facet.Lookup{Id: item.Id, Value: f + j}
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
