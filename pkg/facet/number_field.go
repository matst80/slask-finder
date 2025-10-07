package facet

import (
	"math"

	"github.com/RoaringBitmap/roaring/v2"
	"github.com/matst80/slask-finder/pkg/types"
)

type FieldNumberValue interface {
	int | float64
}

type DecimalField struct {
	*types.BaseField
	*NumberRange[float64]                      // Public min/max in original float domain
	buckets               map[int]*ValueBucket // Coarse bucket -> sorted value entries (integer cents)
	AllValuesCents        map[uint32]int64     // Raw value per item in integer cents
	Count                 int                  `json:"count"`
}

type DecimalFieldResult struct {
	//Count uint    `json:"count,omitempty"`
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

func (k *DecimalFieldResult) HasValues() bool {
	return k.Min < k.Max
}

func (f *DecimalField) GetExtents(matchIds *types.ItemList) *DecimalFieldResult {
	if matchIds == nil {
		return nil
	}
	if f.Count == 0 {
		return &DecimalFieldResult{Min: 0, Max: 0}
	}
	minC := int64(math.MaxInt64)
	maxC := int64(math.MinInt64)

	matchIds.ForEach(func(id uint32) bool {
		if v, ok := f.AllValuesCents[id]; ok {
			if v < minC {
				minC = v
			}
			if v > maxC {
				maxC = v
			}
		}
		return true
	})

	if minC == int64(math.MaxInt64) {
		// no matches had values
		return &DecimalFieldResult{Min: 0, Max: 0}
	}

	return &DecimalFieldResult{
		Min: float64(minC) / 100.0,
		Max: float64(maxC) / 100.0,
	}
}

func (f *DecimalField) IsExcludedFromFacets() bool {
	return f.HideFacet || f.BaseField.InternalOnly
}

func (f *DecimalField) IsCategory() bool {
	return false
}

func (f *DecimalField) MatchesRange(minValue float64, maxValue float64) *types.ItemList {
	if minValue > maxValue {
		return types.NewItemList()
	}
	if f.Count == 0 {
		return types.NewItemList()
	}
	// Full range shortcut
	if minValue <= f.Min && maxValue >= f.Max {
		return nil
	}

	// Convert to integer cents (round outward to be inclusive)
	minC := int64(math.Floor(minValue*100.0 + 0.0000001))
	maxC := int64(math.Ceil(maxValue*100.0 - 0.0000001))
	if minC > maxC {
		return types.NewItemList()
	}

	minBucket := GetBucketFromCents(minC)
	maxBucket := GetBucketFromCents(maxC)

	acc := roaring.NewBitmap()

	// Single bucket case
	if minBucket == maxBucket {
		if b, ok := f.buckets[minBucket]; ok {
			b.RangeUnion(minC, maxC, acc)
		}
		return types.FromBitmap(acc)
	}

	// Partial start bucket
	if b, ok := f.buckets[minBucket]; ok {
		upper := f.bucketUpperBoundCents(minBucket)
		if upper > maxC {
			upper = maxC
		}
		b.RangeUnion(minC, upper, acc)
	}

	// Full middle buckets
	for bId := minBucket + 1; bId < maxBucket; bId++ {
		if b, ok := f.buckets[bId]; ok {
			acc.Or(b.merged)
		}
	}

	// Partial end bucket
	if b, ok := f.buckets[maxBucket]; ok {
		lower := f.bucketLowerBoundCents(maxBucket)
		if lower < minC {
			lower = minC
		}
		b.RangeUnion(lower, maxC, acc)
	}

	return types.FromBitmap(acc)
}

func (f *DecimalField) Match(input any) *types.ItemList {
	value, ok := input.(types.RangeFilter)
	if ok {
		min, minOk := value.Min.(float64)
		max, maxOk := value.Max.(float64)
		if minOk && maxOk {
			return f.MatchesRange(min, max)
		}
	}
	return types.NewItemList()
}

func (f *DecimalField) updateBaseField(field *types.BaseField) {
	f.BaseField.UpdateFrom(field)
}

func (f *DecimalField) UpdateBaseField(field *types.BaseField) {
	f.updateBaseField(field)
}

func (f *DecimalField) MatchAsync(input any, ch chan<- *types.ItemList) {
	ch <- f.Match(input)
}

func (f *DecimalField) GetBaseField() *types.BaseField {
	return f.BaseField
}

type NumberRange[V FieldNumberValue] struct {
	Min V `json:"min"`
	Max V `json:"max"`
}

func (f *DecimalField) Bounds() NumberRange[float64] {
	return *f.NumberRange
}

func (f *DecimalField) GetValues() []any {
	return []any{f.NumberRange}
}

// Helper bounds (coarse bucket) in integer cents domain
func (f *DecimalField) bucketLowerBoundCents(bucket int) int64 {
	return int64(bucket << Bits_To_Shift)
}

func (f *DecimalField) bucketUpperBoundCents(bucket int) int64 {
	// Upper inclusive bound: next bucket start - 1
	return int64(((bucket + 1) << Bits_To_Shift) - 1)
}

func (f *DecimalField) addValueLink(val float64, itemId uint32) bool {
	if !f.Searchable {
		return false
	}

	cents := int64(math.Round(val * 100.0))

	if f.Count == 0 {
		f.Min, f.Max = val, val
	} else {
		if val < f.Min {
			f.Min = val
		} else if val > f.Max {
			f.Max = val
		}
	}
	f.Count++
	f.AllValuesCents[itemId] = cents

	bId := GetBucketFromCents(cents)
	b, ok := f.buckets[bId]
	if !ok {
		b = NewValueBucket()
		f.buckets[bId] = b
	}

	b.AddValue(cents, itemId)
	return true
}

func (f *DecimalField) AddValueLink(data any, id types.ItemId) bool {
	if !f.Searchable {
		return false
	}
	val, ok := data.(float64)
	if !ok {
		return false
	}
	return f.addValueLink(val, uint32(id))
}

func (f *DecimalField) RemoveValueLink(data any, itemId types.ItemId) {
	val, ok := data.(float64)
	if !ok {
		return
	}
	f.removeValueLink(val, uint32(itemId))
}

func (f *DecimalField) removeValueLink(val float64, id uint32) {
	cents := int64(math.Round(val * 100.0))

	delete(f.AllValuesCents, id)
	bId := GetBucketFromCents(cents)
	if b, ok := f.buckets[bId]; ok {
		b.RemoveValue(cents, id)
		if f.Count > 0 {
			f.Count--
		}
	}
}

func (f *DecimalField) TotalCount() int {
	return f.Count
}

func (f *DecimalField) GetRangeForIds(ids *types.IdList) NumberRange[float64] {
	return NumberRange[float64]{Min: f.Min, Max: f.Max}
}

func (DecimalField) GetType() uint {
	return types.FacetNumberType
}

func EmptyDecimalField(field *types.BaseField) *DecimalField {
	return &DecimalField{
		BaseField:      field,
		NumberRange:    &NumberRange[float64]{Min: 0, Max: 0},
		buckets:        make(map[int]*ValueBucket),
		AllValuesCents: make(map[uint32]int64),
		Count:          0,
	}
}
