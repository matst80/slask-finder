package facet

import (
	"maps"
	"unsafe"
)

type IntegerField struct {
	*BaseField
	*NumberRange[int]
	buckets map[int]Bucket[int]
	all     *ItemList
	Count   int `json:"count"`
}

func (f *IntegerField) MatchesRange(minValue int, maxValue int) *ItemList {
	if minValue > maxValue {
		return &ItemList{}
	}
	if minValue <= f.Min && maxValue >= f.Max {
		return f.all
	}
	minBucket := GetBucket(max(minValue, f.Min))
	maxBucket := GetBucket(min(maxValue, f.Max))
	found := make(ItemList, f.Count)

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

func (f IntegerField) Size() int {
	sum := int(unsafe.Sizeof(*f.all))
	for _, bucket := range f.buckets {
		sum += int(unsafe.Sizeof(bucket.values))
	}
	return sum
}

func (f IntegerField) Match(input interface{}) *ItemList {
	value, ok := input.(NumberRange[int])
	if ok {
		return f.MatchesRange(value.Min, value.Max)
	}
	return &ItemList{}
}

func (f IntegerField) GetBaseField() *BaseField {
	return f.BaseField
}

func (f *IntegerField) Bounds() NumberRange[int] {

	return *f.NumberRange
}

func (f IntegerField) GetValues() interface{} {
	return f
}

func (f IntegerField) AddValueLink(data interface{}, item Item) bool {
	value, ok := data.(int)
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

func (f IntegerField) RemoveValueLink(data interface{}, id uint) {
	value, ok := data.(int)
	if !ok {
		return
	}

	bucket := GetBucket(value)
	bucketValues, ok := f.buckets[bucket]
	delete(*f.all, id)
	if ok {
		f.Count--
		bucketValues.RemoveValueLink(value, id)
	}
}

func (f *IntegerField) TotalCount() int {
	return f.Count
}

func (f *IntegerField) GetRangeForIds(ids *IdList) NumberRange[int] {
	return NumberRange[int]{Min: f.Min, Max: f.Max}
}

func (IntegerField) GetType() uint {
	return FacetIntegerType
}

func EmptyIntegerField(field *BaseField) IntegerField {
	return IntegerField{
		BaseField:   field,
		NumberRange: &NumberRange[int]{Min: 0, Max: 0},
		all:         &ItemList{},
		buckets:     map[int]Bucket[int]{},
	}
}
