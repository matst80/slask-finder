package facet

import (
	"fmt"
	"testing"

	"github.com/matst80/slask-finder/pkg/types"
)

type TestItem struct {
	Id uint
}

// func makeItem(id uint) types.Item {
// 	return &types.MockItem{
// 		Id: id,
// 	}
// }

func (i TestItem) GetId() uint {
	return i.Id
}

var total = 300000

func makeNumberField() *IntegerField {
	r := EmptyIntegerField(&types.BaseField{Id: 1, Name: "number", Searchable: true, Description: "number field"})
	for i := range total {
		//for j := 0; j < 100; j++ {
		r.AddValueLink(i, uint(i))
		//r.AddValueLink(i, uint((total*100)+total+i+j))
		//}
	}
	return &r
}

func makeDecimalField() *DecimalField {
	r := EmptyDecimalField(&types.BaseField{Id: 1, Name: "number", Description: "number field"})
	for i := range total {
		for j := range 100 {
			r.AddValueLink(i, uint((total*100)+j))
			r.AddValueLink(i, uint((total*100)+total+i+j))
		}
	}
	return &r
}

var ranges = []struct {
	min int
	max int
}{
	{min: 1, max: total},
	{min: 1, max: 9999},
	{min: 1, max: 999},
	{min: 1, max: 99},
	{min: 1, max: 9},
}

func BenchmarkMatchesRangeFloat(b *testing.B) {
	NumberField := makeDecimalField()
	for _, r := range ranges {
		b.Run(fmt.Sprintf("MatchesRange %d %d", r.min, r.max), func(b *testing.B) {
			NumberField.MatchesRange(float64(r.min), float64(r.max))
		})
	}

}

func BenchmarkMatchesRangeInt(b *testing.B) {
	NumberField := makeNumberField()
	for _, r := range ranges {
		b.Run(fmt.Sprintf("MatchesRange %d %d", r.min, r.max), func(b *testing.B) {
			NumberField.MatchesRange(r.min, r.max)
		})
	}
}

func BenchmarkRangeFunction(b *testing.B) {
	NumberField := makeNumberField()
	ids := &types.ItemList{}
	for id := range NumberField.AllValues {
		ids.AddId(id)
	}
	b.Logf("Extent values %d", len(NumberField.AllValues))

	b.Run(fmt.Sprintf("Extents min %d", total), func(b *testing.B) {

		c := NumberField.GetExtents(ids)
		b.Logf("Extents %d %d, id len: %d", c.Min, c.Max, len(*ids))
	})

	b.Run(fmt.Sprintf("Extents if %d", total), func(b *testing.B) {

		c := NumberField.GetExtents2(ids)
		b.Logf("Extents %d %d, id len: %d", c.Min, c.Max, len(*ids))
	})

}
