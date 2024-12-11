package facet

import (
	"maps"

	"github.com/matst80/slask-finder/pkg/types"
)

type IntegerField struct {
	*types.BaseField
	*NumberRange[int]
	buckets   map[int]Bucket[int]
	allValues map[uint]int
	Count     int `json:"count"`
}

func (f *IntegerField) ValueForItemId(id uint) *int {
	if v, ok := f.allValues[id]; ok {
		return &v
	}
	return nil
}

func (f *IntegerField) MatchesRange(minValue int, maxValue int) *types.ItemList {
	if minValue > maxValue {
		return &types.ItemList{}
	}
	if minValue <= f.Min && maxValue >= f.Max {
		return nil
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

func (f IntegerField) Match(input interface{}) *types.ItemList {
	value, ok := input.(RangeFilter)
	if ok {
		min, minOk := value.Min.(float64)
		max, maxOk := value.Max.(float64)

		if minOk && maxOk {
			return f.MatchesRange(int(min), int(max))
		}
	}

	return &types.ItemList{}
}

func (f IntegerField) GetBaseField() *types.BaseField {
	return f.BaseField
}

func (f *IntegerField) Bounds() NumberRange[int] {

	return *f.NumberRange
}

func (f IntegerField) GetValues() []interface{} {
	return []interface{}{f.NumberRange}
}

func (f IntegerField) addValueLink(value int, item types.Item) {
	f.Min = min(f.Min, value)
	f.Max = max(f.Max, value)
	f.Count++
	bucket := GetBucket(value)
	bucketValues, ok := f.buckets[bucket]
	f.allValues[item.GetId()] = value
	if !ok {
		f.buckets[bucket] = MakeBucket(value, item)
	} else {
		bucketValues.AddValueLink(value, item)
	}
}

func (f IntegerField) AddValueLink(data interface{}, item types.Item) bool {
	switch value := data.(type) {
	case int:
		f.addValueLink(value, item)
		return true
	case float64:
		f.addValueLink(int(value), item)
		return true
	}

	return false
}

func (f *IntegerField) removeValueLink(value int, id uint) {
	bucket := GetBucket(value)
	bucketValues, ok := f.buckets[bucket]
	delete(f.allValues, id)
	if ok {
		f.Count--
		bucketValues.RemoveValueLink(value, id)
	}
}

func (f IntegerField) RemoveValueLink(data interface{}, id uint) {
	switch value := data.(type) {
	case int:
		f.removeValueLink(value, id)
	case float64:
		f.removeValueLink(int(value), id)
	}
}

func (f *IntegerField) TotalCount() int {
	return f.Count
}

// func (f *IntegerField) GetRangeForIds(ids *IdList) NumberRange[int] {
// 	return NumberRange[int]{Min: f.Min, Max: f.Max}
// }

func (IntegerField) GetType() uint {
	return types.FacetIntegerType
}

func EmptyIntegerField(field *types.BaseField) IntegerField {
	return IntegerField{
		BaseField:   field,
		allValues:   map[uint]int{},
		NumberRange: &NumberRange[int]{Min: 0, Max: 0},
		buckets:     map[int]Bucket[int]{},
	}
}
