package facet

import (
	"fmt"
	"testing"

	"github.com/matst80/slask-finder/pkg/types"
)

type TestItem struct {
	Id uint
}

func makeItem(id uint) types.Item {
	return &types.MockItem{
		Id: id,
	}
}

func (i TestItem) GetId() uint {
	return i.Id
}

var total = 20000

func makeNumberField() *IntegerField {
	r := EmptyIntegerField(&types.BaseField{Id: 1, Name: "number", Description: "number field"})
	for i := 0; i < total; i++ {
		for j := 0; j < 100; j++ {
			r.AddValueLink(i, uint((total*100)+j))
			r.AddValueLink(i, uint((total*100)+total+i+j))
		}
	}
	return &r
}

func makeDecimalField() *DecimalField {
	r := EmptyDecimalField(&types.BaseField{Id: 1, Name: "number", Description: "number field"})
	for i := 0; i < total; i++ {
		for j := 0; j < 100; j++ {
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
