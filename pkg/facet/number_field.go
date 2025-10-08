package facet

import (
	"log"
	"math"
	"strconv"

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
	// Preserve legacy semantics: if caller passed nil, we return nil (caller checks).
	if matchIds == nil {
		return nil
	}
	// No values indexed
	if f.Count == 0 {
		return &DecimalFieldResult{Min: 0, Max: 0}
	}
	// Empty filter => no results
	if matchIds.Len() == 0 {
		return &DecimalFieldResult{Min: 0, Max: 0}
	}
	// Full coverage (all items that have a value)
	if int(matchIds.Cardinality()) == f.Count {
		return &DecimalFieldResult{Min: f.Min, Max: f.Max}
	}

	bm := matchIds.Bitmap()
	if bm == nil || bm.IsEmpty() {
		return &DecimalFieldResult{Min: 0, Max: 0}
	}

	// Recover bucket span using the stored min/max (convert to the same cents representation used in buckets)
	minCentsField := int64(math.Round(f.Min * 100.0))
	maxCentsField := int64(math.Round(f.Max * 100.0))
	startBucket := GetBucketFromCents(minCentsField)
	endBucket := GetBucketFromCents(maxCentsField)

	// Use shared helpers to scan for min/max (values stored as integer cents)
	minC, okMin := ScanMinValue(f.buckets, startBucket, endBucket, bm)
	if !okMin {
		return &DecimalFieldResult{Min: 0, Max: 0}
	}
	maxC, okMax := ScanMaxValue(f.buckets, startBucket, endBucket, bm)
	if !okMax {
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
	if minValue > maxValue || f.Count == 0 {
		return types.NewItemList()
	}
	// Full range shortcut (return nil sentinel meaning "all")
	if minValue <= f.Min && maxValue >= f.Max {
		return nil
	}
	// Clamp to stored extents to avoid unnecessary bucket scans
	if minValue < f.Min {
		minValue = f.Min
	}
	if maxValue > f.Max {
		maxValue = f.Max
	}
	// Inclusive outward rounding to cents (matching previous logic)
	minC := int64(math.Floor(minValue*100.0 + 0.0000001))
	maxC := int64(math.Ceil(maxValue*100.0 - 0.0000001))
	if minC > maxC {
		return types.NewItemList()
	}
	minBucket := GetBucketFromCents(minC)
	maxBucket := GetBucketFromCents(maxC)

	// Single bucket fast path
	if minBucket == maxBucket {
		acc := roaring.NewBitmap()
		if b, ok := f.buckets[minBucket]; ok {
			b.RangeUnion(minC, maxC, acc)
		}
		return types.FromBitmap(acc)
	}

	acc := roaring.NewBitmap()

	// Partial start bucket
	if b, ok := f.buckets[minBucket]; ok {
		upper := BucketUpperBound(minBucket)
		if upper > maxC {
			upper = maxC
		}
		b.RangeUnion(minC, upper, acc)
	}

	// Middle buckets batched OR
	if maxBucket > minBucket+1 {
		mids := make([]*roaring.Bitmap, 0, (maxBucket-minBucket)-1)
		for bId := minBucket + 1; bId < maxBucket; bId++ {
			if b, ok := f.buckets[bId]; ok && b.merged != nil && !b.merged.IsEmpty() {
				mids = append(mids, b.merged)
			}
		}
		if len(mids) == 1 {
			acc.Or(mids[0])
		} else if len(mids) > 1 {
			acc.Or(roaring.FastOr(mids...))
		}
	}

	// Partial end bucket
	if b, ok := f.buckets[maxBucket]; ok {
		lower := BucketLowerBound(maxBucket)
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

// (bucket bound helpers moved to number_helpers.go)

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

func (f *DecimalField) AddValueLink(data any, itemId types.ItemId) bool {
	if !f.Searchable {
		return false
	}
	id := uint32(itemId)
	switch value := data.(type) {
	case int:
		f.addValueLink(float64(value), id)
		return true
	case float64:
		f.addValueLink(value, id)
		return true
	case string:
		floatValue, err := strconv.ParseFloat(value, 64)
		if err == nil {
			f.addValueLink(float64(floatValue), id)
			return true
		}
	case []string:
		if len(value) > 0 {
			first := value[0]
			floatValue, err := strconv.ParseFloat(first, 64)
			if err == nil {
				f.addValueLink(float64(floatValue), id)
				return true
			}
		}
	default:
		log.Printf("'%v': AddValueLink: %T %d (%s)", data, value, f.Id, f.Name)
	}
	return false
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
