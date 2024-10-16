package index

import (
	"fmt"
	"sort"
	"strings"

	"tornberg.me/facet-search/pkg/facet"
)

type SortingData struct {
	price    int
	orgPrice int
	grade    int
	noGrades int
	sellable bool
	margin   float64
}

func getSortingData(item *DataItem, data *SortingData) {
	data.price = 0
	data.orgPrice = 0
	data.grade = 0
	data.noGrades = 0

	for _, f := range item.IntegerFields {
		if f.Id == 4 {
			data.price = f.Value
		}
		if f.Id == 5 {
			data.orgPrice = f.Value
		}
		if f.Id == 6 {
			data.grade = f.Value
		}
		if f.Id == 7 {
			data.noGrades = f.Value
		}
	}
	//return SortingData{price, orgPrice, grade, noGrades, (item.Buyable || item.BuyableInStore), item.MarginPercent}
}

func getPopularValue(itemData *SortingData, overrideValue float64) float64 {
	v := (overrideValue * 1000.0)
	if itemData.orgPrice > 0 && itemData.orgPrice-itemData.price > 0 {
		discount := itemData.orgPrice - itemData.price
		v += ((float64(discount) / float64(itemData.orgPrice)) * 100000.0) + (float64(discount) / 5.0)
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
		v += 700
	}
	v += itemData.margin * 400
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

type StaticPositions map[int]uint

func (s *StaticPositions) ToString() string {
	ret := ""
	for key, value := range *s {
		ret += fmt.Sprintf("%d:%d,", key, value)
	}
	return ret
}

func (s *StaticPositions) FromString(data string) error {
	*s = make(map[int]uint)
	for _, item := range strings.Split(data, ",") {
		var key int
		var value uint
		_, err := fmt.Sscanf(item, "%d:%d", &key, &value)
		if err != nil {
			if err.Error() == "EOF" {
				return nil
			}
			return err
		}
		(*s)[key] = value
	}
	return nil
}

type SortOverride map[uint]float64

func (s *SortOverride) ToString() string {
	ret := ""
	for key, value := range *s {
		ret += fmt.Sprintf("%d:%f,", key, value)
	}
	return ret
}

func (s *SortOverride) FromString(data string) error {
	*s = make(map[uint]float64)
	for _, item := range strings.Split(data, ",") {
		var key uint
		var value float64
		_, err := fmt.Sscanf(item, "%d:%f", &key, &value)
		if err != nil {
			if err.Error() == "EOF" {
				return nil
			}
			return err
		}
		(*s)[key] = value
	}
	return nil
}
