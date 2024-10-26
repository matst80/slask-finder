package facet

import (
	"maps"
	"unsafe"

	"github.com/matst80/slask-finder/pkg/types"
)

type FieldNumberValue interface {
	int | float64
}

type DecimalField struct {
	*types.BaseField
	*NumberRange[float64]
	buckets map[int]Bucket[float64]
	all     *types.ItemList
	Count   int `json:"count"`
}

func (f *DecimalField) MatchesRange(minValue float64, maxValue float64) *types.ItemList {
	if minValue > maxValue {
		return &types.ItemList{}
	}
	if minValue <= f.Min && maxValue >= f.Max {
		return f.all
	}
	minBucket := GetBucket(max(minValue, f.Min))
	maxBucket := GetBucket(min(maxValue, f.Max))
	found := make(types.ItemList, f.Count)

	for v, ids := range f.buckets[minBucket].values {
		if v >= minValue && v <= maxValue {
			maps.Copy(found, ids)
		}
	}

	if minBucket < maxBucket {

		for id := minBucket + 1; id < maxBucket; id++ {
			if bucket, ok := f.buckets[id]; ok {
				for _, ids := range bucket.values {
					maps.Copy(found, ids)
				}
				//maps.Copy(found, *bucket.all)
			}

		}

		for v, ids := range f.buckets[maxBucket].values {
			if v <= maxValue {
				maps.Copy(found, ids)
			}
		}

	}
	return &found
}

func (f DecimalField) Size() int {
	return int(unsafe.Sizeof(f.buckets)) + int(unsafe.Sizeof(f.all))
}

func (f DecimalField) Match(input interface{}) *types.ItemList {
	value, ok := input.(NumberRange[float64])
	if ok {
		return f.MatchesRange(value.Min, value.Max)
	}
	return &types.ItemList{}
}

func (f DecimalField) GetBaseField() *types.BaseField {
	return f.BaseField
}

type NumberRange[V FieldNumberValue] struct {
	Min V `json:"min"`
	Max V `json:"max"`
}

func (f *DecimalField) Bounds() NumberRange[float64] {
	return *f.NumberRange
}

func (f DecimalField) GetValues() []interface{} {
	return []interface{}{f.NumberRange}
}

func (f DecimalField) AddValueLink(data interface{}, item types.Item) bool {
	value, ok := data.(float64)
	if !ok {
		return false
	}

	f.Min = min(f.Min, value)
	f.Max = max(f.Max, value)
	f.Count++
	f.all.Add(item)
	bucket := GetBucket(value)
	bucketValues, ok := f.buckets[bucket]
	if !ok {
		f.buckets[bucket] = MakeBucket(value, item)
	} else {
		bucketValues.AddValueLink(value, item)
	}
	return true
}

func (f DecimalField) RemoveValueLink(data interface{}, id uint) {
	value, ok := data.(float64)
	if !ok {
		return
	}

	bucket := GetBucket(value)
	bucketValues, ok := f.buckets[bucket]
	delete(*f.all, id)
	if ok {
		(&f).Count--
		bucketValues.RemoveValueLink(value, id)
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

func EmptyDecimalField(field *types.BaseField) DecimalField {
	return DecimalField{
		BaseField:   field,
		NumberRange: &NumberRange[float64]{Min: 0, Max: 0},
		all:         &types.ItemList{},
		buckets:     map[int]Bucket[float64]{},
	}
}
