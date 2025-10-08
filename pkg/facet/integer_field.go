package facet

import (
	"log"
	"strconv"

	"github.com/RoaringBitmap/roaring/v2"
	"github.com/matst80/slask-finder/pkg/types"
)

type IntegerFieldResult struct {
	//	Count   uint   `json:"count,omitempty"`
	Min     int    `json:"min"`
	Max     int    `json:"max"`
	Buckets []uint `json:"buckets,omitempty"`
}

func (k *IntegerFieldResult) HasValues() bool {
	return k.Min < k.Max
}

const (
	EXPECTED_RESULT_SIZE = 20
	MAX_RESULT_VALUE     = float64(100)
)

func NormalizeValues(input []uint) []uint {
	min := uint(0)
	max := uint(0)
	for _, v := range input {
		if v < min {
			min = v
		} else if v > max {
			max = v
		}
	}
	result := make([]uint, len(input))
	for i := range input {
		result[i] = uint((float64(input[i]-min) / float64(max-min)) * MAX_RESULT_VALUE)
	}
	return result
}

func NormalizeResults(input []uint) []uint {
	l := len(input)
	if l <= EXPECTED_RESULT_SIZE {
		return NormalizeValues(input)
	}

	result := make([]uint, 0, EXPECTED_RESULT_SIZE)
	itemsToGroup := l / EXPECTED_RESULT_SIZE
	sum := uint(0)
	for i := range l {
		sum += input[i]
		if (i+1)%itemsToGroup == 0 {
			result = append(result, sum)
			sum = 0
		}
	}
	return NormalizeValues(result)
}

type IntegerField struct {
	*types.BaseField
	*NumberRange[int]
	buckets   map[int]*ValueBucket // optimized bucket structure (integer domain already)
	AllValues map[uint32]int
	Count     int `json:"count"`
}

func (f *IntegerField) IsExcludedFromFacets() bool {
	return f.HideFacet || f.BaseField.InternalOnly
}

func (f *IntegerField) IsCategory() bool {
	return false
}

func (f *IntegerField) GetExtents(matchIds *types.ItemList) *IntegerFieldResult {
	// No values stored
	if f.Count == 0 {
		return &IntegerFieldResult{Min: 0, Max: 0}
	}
	// Nil or empty means "no filtering" -> full extents
	if matchIds == nil || matchIds.Len() == 0 {
		return &IntegerFieldResult{Min: f.Min, Max: f.Max}
	}
	// Full coverage (all known value items included)
	if int(matchIds.Cardinality()) == f.Count {
		return &IntegerFieldResult{Min: f.Min, Max: f.Max}
	}

	bm := matchIds.Bitmap()
	if bm == nil || bm.IsEmpty() {
		return &IntegerFieldResult{Min: 0, Max: 0}
	}

	startBucket := GetBucket(f.Min)
	endBucket := GetBucket(f.Max)

	// Use shared helpers to scan for min/max across buckets
	minValue, okMin := ScanMinValue(f.buckets, startBucket, endBucket, bm)
	if !okMin {
		return &IntegerFieldResult{Min: 0, Max: 0}
	}
	maxValue, okMax := ScanMaxValue(f.buckets, startBucket, endBucket, bm)
	if !okMax {
		return &IntegerFieldResult{Min: 0, Max: 0}
	}
	return &IntegerFieldResult{Min: int(minValue), Max: int(maxValue)}
}

func (f *IntegerField) ValueForItemId(id uint32) *int {
	if v, ok := f.AllValues[id]; ok {
		return &v
	}
	return nil
}

func (f *IntegerField) GetBucketSizes(minValue int, maxValue int) []uint {
	if minValue > maxValue || f.Count == 0 {
		return []uint{}
	}
	minBucket := GetBucket(max(minValue, f.Min))
	maxBucket := GetBucket(min(maxValue, f.Max))
	if maxBucket < minBucket {
		return []uint{}
	}
	sizes := make([]uint, maxBucket-minBucket+1)
	for b := minBucket; b <= maxBucket; b++ {
		if vb, ok := f.buckets[b]; ok {
			// merged cardinality is closer to distinct items in bucket
			sizes[b-minBucket] = uint(vb.merged.GetCardinality())
		}
	}
	return sizes
}

func (f *IntegerField) MatchesRange(minValue int, maxValue int) *types.ItemList {
	if minValue > maxValue || f.Count == 0 {
		return types.NewItemList()
	}
	// Full coverage shortcut
	if minValue <= f.Min && maxValue >= f.Max {
		return nil
	}
	// Clamp
	if minValue < f.Min {
		minValue = f.Min
	}
	if maxValue > f.Max {
		maxValue = f.Max
	}

	minBucket := GetBucket(minValue)
	maxBucket := GetBucket(maxValue)

	// Single bucket fast path
	if minBucket == maxBucket {
		acc := roaring.NewBitmap()
		if b, ok := f.buckets[minBucket]; ok {
			b.RangeUnion(int64(minValue), int64(maxValue), acc)
		}
		return types.FromBitmap(acc)
	}

	acc := roaring.NewBitmap()

	// Partial start bucket
	if b, ok := f.buckets[minBucket]; ok {
		upper := BucketUpperBound(minBucket)
		if upper > int64(maxValue) {
			upper = int64(maxValue)
		}
		b.RangeUnion(int64(minValue), upper, acc)
	}

	// Middle buckets (batch OR)
	if maxBucket > minBucket+1 {
		mids := make([]*roaring.Bitmap, 0, (maxBucket-minBucket)-1)
		for id := minBucket + 1; id < maxBucket; id++ {
			if b, ok := f.buckets[id]; ok && b.merged != nil && !b.merged.IsEmpty() {
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
		if lower < int64(minValue) {
			lower = int64(minValue)
		}
		b.RangeUnion(lower, int64(maxValue), acc)
	}

	return types.FromBitmap(acc)
}

func (f *IntegerField) Match(input any) *types.ItemList {
	value, ok := input.(types.RangeFilter)
	if ok {
		min, minOk := value.Min.(float64)
		max, maxOk := value.Max.(float64)

		if minOk && maxOk {
			return f.MatchesRange(int(min), int(max))
		}
	}

	return types.NewItemList()
}

func (f *IntegerField) UpdateBaseField(field *types.BaseField) {
	f.BaseField.UpdateFrom(field)
}

func (f *IntegerField) MatchAsync(input any, ch chan<- *types.ItemList) {
	ch <- f.Match(input)
}

func (f *IntegerField) GetBaseField() *types.BaseField {
	return f.BaseField
}

func (f *IntegerField) Bounds() NumberRange[int] {
	return *f.NumberRange
}

func (f *IntegerField) GetValues() []any {
	return []any{f.NumberRange}
}

func (f *IntegerField) addValueLink(value int, itemId uint32) {
	if f.Count == 0 {
		f.Min, f.Max = value, value
	} else {
		if value < f.Min {
			f.Min = value
		}
		if value > f.Max {
			f.Max = value
		}
	}
	f.Count++
	f.AllValues[itemId] = value
	bId := GetBucket(value)
	b, ok := f.buckets[bId]
	if !ok {
		b = NewValueBucket()
		f.buckets[bId] = b
	}
	b.AddValue(int64(value), itemId)
}

func (f *IntegerField) AddValueLink(data any, itemId types.ItemId) bool {
	if !f.Searchable {
		return false
	}
	id := uint32(itemId)
	switch value := data.(type) {
	case int:
		f.addValueLink(value, id)
		return true
	case float64:
		f.addValueLink(int(value), id)
		return true
	case string:
		intValue, err := strconv.Atoi(value)
		if err == nil {
			f.addValueLink(intValue, id)
			return true
		}
	case []string:
		if len(value) > 0 {
			first := value[0]
			intValue, err := strconv.Atoi(first)
			if err == nil {
				f.addValueLink(intValue, id)
				return true
			}
		}
	default:
		log.Printf("'%v': AddValueLink: %T %d (%s)", data, value, f.Id, f.Name)
	}

	return false
}

func (f *IntegerField) removeValueLink(value int, id uint32) {
	delete(f.AllValues, id)
	bId := GetBucket(value)
	if b, ok := f.buckets[bId]; ok {
		b.RemoveValue(int64(value), id)
		if f.Count > 0 {
			f.Count--
		}
		// Lazy: not rebuilding merged; acceptable for low deletion rate.
	}
}

func (f *IntegerField) RemoveValueLink(data any, itemId types.ItemId) {
	id := uint32(itemId)
	switch value := data.(type) {
	case int:
		f.removeValueLink(value, id)
	case float64:
		f.removeValueLink(int(value), id)
	case string:
		intValue, err := strconv.Atoi(value)
		if err == nil {
			f.removeValueLink(intValue, id)
		}
	case []string:
		if len(value) > 0 {
			if intValue, err := strconv.Atoi(value[0]); err == nil {
				f.removeValueLink(intValue, id)
			}
		}
	}
}

func (f *IntegerField) TotalCount() int {
	return f.Count
}

func (IntegerField) GetType() uint {
	return types.FacetIntegerType
}

func EmptyIntegerField(field *types.BaseField) *IntegerField {
	return &IntegerField{
		BaseField:   field,
		AllValues:   map[uint32]int{},
		NumberRange: &NumberRange[int]{Min: 0, Max: 0},
		buckets:     map[int]*ValueBucket{},
	}
}
