package index

import (
	"sort"

	"tornberg.me/facet-search/pkg/facet"
)

type SortingData struct {
	price    int
	orgPrice int
	grade    int
	noGrades int
	sellable bool
}

func getSortingData(item *DataItem) SortingData {
	price := 0
	orgPrice := 0
	grade := 0
	noGrades := 0

	for _, f := range item.IntegerFields {
		if f.Id == 4 {
			price = f.Value
		}
		if f.Id == 5 {
			orgPrice = f.Value
		}
		if f.Id == 6 {
			grade = f.Value
		}
		if f.Id == 7 {
			noGrades = f.Value
		}
	}
	return SortingData{price, orgPrice, grade, noGrades, item.SaleStatus == "ACT"}
}

func getPopularValue(itemData SortingData, overrideValue float64) float64 {
	v := overrideValue
	if itemData.orgPrice > 0 && itemData.orgPrice-itemData.price > 0 {
		discount := itemData.orgPrice - itemData.price
		v += float64(discount/itemData.orgPrice) * 1000000
	}
	if itemData.sellable {
		v += 5000
	}
	if itemData.price > 99999900 {
		v -= 2500
	}
	if itemData.price < 10000 {
		v -= 800
	}
	if itemData.price%900 == 0 {
		v += float64(10000 - (itemData.price / 10000))
	}
	return v + float64(itemData.grade*itemData.noGrades)
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
