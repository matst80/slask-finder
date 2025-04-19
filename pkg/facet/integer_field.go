package facet

import (
	"maps"
	"strconv"

	"github.com/matst80/slask-finder/pkg/types"
)

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
	for i := 0; i < len(input); i++ {
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
	for i := 0; i < l; i++ {
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

func (f *IntegerField) GetBucketSizes(minValue int, maxValue int) []uint {
	if minValue > maxValue {
		return []uint{}
	}
	minBucket := GetBucket(max(minValue, f.Min))
	maxBucket := GetBucket(min(maxValue, f.Max))
	bucketSizes := make([]uint, maxBucket-minBucket+1)
	for i := minBucket; i <= maxBucket; i++ {
		if bucket, ok := f.buckets[i]; ok {
			bucketSizes[i-minBucket] = uint(len(bucket.values))
		}
	}
	return bucketSizes
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
	value, ok := input.(types.RangeFilter)
	if ok {
		min, minOk := value.Min.(float64)
		max, maxOk := value.Max.(float64)

		if minOk && maxOk {
			return f.MatchesRange(int(min), int(max))
		}
	}

	return &types.ItemList{}
}

func (f IntegerField) MatchAsync(input interface{}, ch chan<- *types.ItemList) {
	ch <- f.Match(input)
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

func (f IntegerField) addValueLink(value int, itemId uint) {
	f.Min = min(f.Min, value)
	f.Max = max(f.Max, value)
	f.Count++
	bucket := GetBucket(value)
	bucketValues, ok := f.buckets[bucket]
	f.allValues[itemId] = value
	if !ok {
		f.buckets[bucket] = MakeBucket(value, itemId)
	} else {
		bucketValues.AddValueLink(value, itemId)
	}
}

func (f IntegerField) AddValueLink(data interface{}, itemId uint) bool {
	if !f.Searchable {
		return false
	}
	switch value := data.(type) {
	case int:
		f.addValueLink(value, itemId)
		return true
	case float64:
		f.addValueLink(int(value), itemId)
		return true
	case string:
		intValue, err := strconv.Atoi(value)
		if err == nil {
			f.addValueLink(intValue, itemId)
			return true
		}
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
	case string:
		intValue, err := strconv.Atoi(value)
		if err == nil {
			f.removeValueLink(intValue, id)
		}
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
