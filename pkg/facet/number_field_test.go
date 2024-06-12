package facet

import (
	"fmt"
	"testing"
	"time"
)

var total = 20000

func makeNumberField[K float64 | int]() *NumberField[K] {
	r := EmptyNumberField[K](&BaseField{Id: 1, Name: "number", Description: "number field"})
	for i := 0; i < total; i++ {
		for j := 0; j < 100; j++ {
			r.AddValueLink(K(i), uint((total*100)+j))
			r.AddValueLink(K(i), uint((total*100)+total+i+j))
		}
	}
	return r
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
	NumberField := makeNumberField[float64]()
	for _, r := range ranges {
		b.Run(fmt.Sprintf("MatchesRange %d %d", r.min, r.max), func(b *testing.B) {

			start := time.Now()
			NumberField.MatchesRange(float64(r.min), float64(r.max))
			b.Logf("Float took %v", time.Since(start))
			//b.Logf("Result: %v", len(res))
		})
	}

}

func BenchmarkMatchesRangeInt(b *testing.B) {
	NumberField := makeNumberField[int]()
	for _, r := range ranges {
		b.Run(fmt.Sprintf("MatchesRange %d %d", r.min, r.max), func(b *testing.B) {

			start := time.Now()
			NumberField.MatchesRange(r.min, r.max)
			b.Logf("Int took %v", time.Since(start))

		})
	}
}
