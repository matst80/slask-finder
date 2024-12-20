package types

type RatingRule struct {
	Multiplier     float64 `json:"multiplier,omitempty"`
	SubtractValue  int     `json:"subtractValue,omitempty"`
	ValueIfNoMatch float64 `json:"valueIfNoMatch,omitempty"`
}

func (r *RatingRule) Type() RuleType {
	return "RatingRule"
}

func (r *RatingRule) New() JsonType {
	return &RatingRule{}
}

func (r *RatingRule) GetValue(item Item) float64 {

	avg, num := item.GetRating()
	if r.Multiplier == 0 {
		r.Multiplier = 1
	}
	if num == 0 {
		return r.ValueIfNoMatch
	} else {
		return (float64(avg) - float64(r.SubtractValue)) * r.Multiplier
	}
}
