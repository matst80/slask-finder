package types

type NumberComparator string

type Source string

type RuleSource struct {
	Source       Source `json:"source"`
	PropertyName string `json:"property,omitempty"`
	FieldId      uint   `json:"fieldId,omitempty"`
}

func (r RuleSource) GetSourceValue(item Item) interface{} {
	fetchByFieldId := r.Source == FieldId || (r.Source == "" && r.FieldId > 0)

	if fetchByFieldId {
		return item.GetFields()[r.FieldId]
	} else {
		if r.PropertyName == "" {
			return nil
		}
		return item.GetPropertyValue(r.PropertyName) // GetPropertyValue(item, r.PropertyName)
	}
}

type NumberLimitRule struct {
	RuleSource
	Limit           float64          `json:"limit"`
	Comparator      NumberComparator `json:"comparator"`
	ValueIfMatch    float64          `json:"value"`
	ValueIfNotMatch float64          `json:"valueIfNotMatch"`
}

const (
	Over     NumberComparator = ">"
	Under    NumberComparator = "<"
	Equal    NumberComparator = "="
	Property Source           = "property"
	FieldId  Source           = "fieldId"
)

func (_ *NumberLimitRule) Type() RuleType {
	return "NumberLimitRule"
}

func (_ *NumberLimitRule) New() JsonType {
	return &NumberLimitRule{}
}

func CompareFactory[K int | float64 | int64](comparator NumberComparator, limit K) func(nr K) bool {
	return func(nr K) bool {
		switch comparator {
		case Over:
			return nr > limit
		case Under:
			return nr < limit
		case Equal:
			return nr == limit
		}
		return false
	}
}

func (r *NumberLimitRule) GetValue(item Item) float64 {

	matchFn := CompareFactory(r.Comparator, r.Limit)
	value := r.GetSourceValue(item)

	v, found := AsNumber[float64](value)

	if !found {
		return r.ValueIfNotMatch
	} else {
		if matchFn(v) {
			return r.ValueIfMatch
		} else {
			return r.ValueIfNotMatch
		}
	}
}
