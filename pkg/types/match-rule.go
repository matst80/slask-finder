package types

type MatchRule struct {
	RuleSource
	Match           any     `json:"match"`
	Invert          bool    `json:"invert,omitempty"`
	ValueIfMatch    float64 `json:"value"`
	ValueIfNotMatch float64 `json:"valueIfNotMatch"`
}

func (_ *MatchRule) Type() RuleType {
	return "MatchRule"
}

func (_ *MatchRule) New() JsonType {
	return &MatchRule{}
}

func (r *MatchRule) GetValue(item Item) float64 {
	match := false
	value := r.GetSourceValue(item)

	if r.Invert {
		match = value != r.Match
	} else {
		match = value == r.Match
	}

	if match {
		return r.ValueIfMatch
	} else {
		return r.ValueIfNotMatch
	}
}
