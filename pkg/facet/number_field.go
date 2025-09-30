package facet

import (
	"maps"

	"github.com/matst80/slask-finder/pkg/types"
)

type FieldNumberValue interface {
	int | float64
}

type DecimalField struct {
	*types.BaseField
	*NumberRange[float64]
	buckets   map[int]Bucket[float64]
	AllValues map[uint]float64
	//all     *types.ItemList
	Count int `json:"count"`
}

type DecimalFieldResult struct {
	//Count uint    `json:"count,omitempty"`
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

func (k *DecimalFieldResult) HasValues() bool {
	return k.Min < k.Max
}

func (f DecimalField) GetExtents(matchIds *types.ItemList) *DecimalFieldResult {
	if matchIds == nil {
		return nil
	}
	minV := f.Max
	maxV := f.Min
	for id := range *matchIds {
		if v, ok := f.AllValues[id]; ok {
			if v < minV {
				minV = v
			} else if v > maxV {
				maxV = v
			}
		}
	}
	return &DecimalFieldResult{
		Min: minV,
		Max: maxV,
	}
}

// func (f *DecimalField) ValueForItemId(id uint) *float64 {
// 	if v, ok := f.AllValues[id]; ok {
// 		return &v
// 	}
// 	return nil
// }

func (f DecimalField) IsExcludedFromFacets() bool {
	return f.HideFacet || f.BaseField.InternalOnly
}

func (f DecimalField) IsCategory() bool {
	return false
}

func (f *DecimalField) MatchesRange(minValue float64, maxValue float64) *types.ItemList {
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

func (f DecimalField) Match(input any) *types.ItemList {
	value, ok := input.(types.RangeFilter)
	if ok {
		min, minOk := value.Min.(float64)
		max, maxOk := value.Max.(float64)

		if minOk && maxOk {
			return f.MatchesRange(min, max)
		}
	}

	return &types.ItemList{}
}

func (f *DecimalField) updateBaseField(field *types.BaseField) {
	f.BaseField.UpdateFrom(field)
}

func (f DecimalField) UpdateBaseField(field *types.BaseField) {
	f.updateBaseField(field)
}

func (f DecimalField) MatchAsync(input any, ch chan<- *types.ItemList) {
	ch <- f.Match(input)
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

func (f DecimalField) GetValues() []any {
	return []any{f.NumberRange}
}

func (f DecimalField) AddValueLink(data any, itemId uint) bool {
	if !f.Searchable {
		return false
	}
	value, ok := data.(float64)
	if !ok {
		return false
	}

	f.Min = min(f.Min, value)
	f.Max = max(f.Max, value)
	f.Count++
	f.AllValues[itemId] = value

	bucket := GetBucket(value)
	bucketValues, ok := f.buckets[bucket]
	if !ok {
		f.buckets[bucket] = MakeBucket(value, itemId)
	} else {
		bucketValues.AddValueLink(value, itemId)
	}
	return true
}

func (f DecimalField) RemoveValueLink(data any, id uint) {
	value, ok := data.(float64)
	if !ok {
		return
	}

	bucket := GetBucket(value)
	bucketValues, ok := f.buckets[bucket]
	delete(f.AllValues, id)
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
		AllValues:   map[uint]float64{},
		BaseField:   field,
		NumberRange: &NumberRange[float64]{Min: 0, Max: 0},
		buckets:     map[int]Bucket[float64]{},
	}
}
