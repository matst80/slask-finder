package facet

import (
	"maps"
)

type FieldNumberValue interface {
	int | float64
}

type NumberField[V FieldNumberValue] struct {
	*BaseField
	buckets map[int]Bucket[V]
	Count   int
	Min     V
	Max     V
}

func (f *NumberField[V]) MatchesRange(minValue V, maxValue V) *IdList {

	minBucket := GetBucket(max(minValue, f.Min))
	maxBucket := GetBucket(min(maxValue, f.Max))
	found := make(IdList, f.Count)

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

type NumberRange[V FieldNumberValue] struct {
	Min V `json:"min"`
	Max V `json:"max"`
}

func (f *NumberField[V]) Bounds() NumberRange[V] {

	return NumberRange[V]{Min: f.Min, Max: f.Max}
}

func (f *NumberField[V]) AddValueLink(value V, id uint) {
	bucket := GetBucket(value)
	bucketValues, ok := f.buckets[bucket]
	f.Min = min(f.Min, value)
	f.Max = max(f.Max, value)
	f.Count++
	if !ok {
		f.buckets[bucket] = MakeBucket(value, id)
	} else {
		bucketValues.AddValueLink(value, id)
	}
}

func (f *NumberField[V]) RemoveValueLink(value V, id uint) {
	bucket := GetBucket(value)
	bucketValues, ok := f.buckets[bucket]

	if ok {
		f.Count--
		bucketValues.RemoveValueLink(value, id)
	}
}

func (f *NumberField[V]) TotalCount() int {
	return f.Count
}

func (f *NumberField[V]) GetRangeForIds(ids *IdList) NumberRange[V] {
	return NumberRange[V]{Min: f.Min, Max: f.Max}
}

func NewNumberField[V FieldNumberValue](field *BaseField, value V, ids *IdList) *NumberField[V] {
	return &NumberField[V]{
		BaseField: field,
		buckets:   map[int]Bucket[V]{GetBucket(value): MakeBucketList(value, ids)},
	}
}

func EmptyNumberField[V FieldNumberValue](field *BaseField) *NumberField[V] {
	return &NumberField[V]{
		BaseField: field,
		buckets:   map[int]Bucket[V]{},
	}
}
