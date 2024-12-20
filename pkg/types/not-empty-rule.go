package types

type NotEmptyRule struct {
	RuleSource
	ValueIfMatch    float64 `json:"value,omitempty"`
	ValueIfNotMatch float64 `json:"valueIfNotMatch"`
}

func (_ *NotEmptyRule) Type() RuleType {
	return "NotEmptyRule"
}

func (_ *NotEmptyRule) New() JsonType {
	return &NotEmptyRule{}
}

func (r *NotEmptyRule) GetValue(item Item) float64 {
	value := r.GetSourceValue(item)
	match := false
	if value == nil {
		return r.ValueIfNotMatch
	}
	switch v := value.(type) {
	case string:
		if v != "" {
			match = true
		}
	case float64:
		if v != 0 {
			match = true
		}
	case int:
		if v > 0 {
			match = true
		}
	}
	if match {
		return r.ValueIfMatch
	}

	return r.ValueIfNotMatch

}
