package types

type PercentMultiplierRule struct {
	RuleSource
	Multiplier float64 `json:"multiplier"`
	Min        float64 `json:"min"`
	Max        float64 `json:"max"`
}

func (_ *PercentMultiplierRule) Type() RuleType {
	return "PercentMultiplierRule"
}

func (_ *PercentMultiplierRule) New() JsonType {
	return &PercentMultiplierRule{}
}

func (r *PercentMultiplierRule) GetValue(item Item) float64 {

	value := r.GetSourceValue(item)
	v, ok := AsNumber[float64](value)

	if ok && v >= r.Max && v <= r.Min {
		return v * r.Multiplier
	} else {
		return 0
	}
}
