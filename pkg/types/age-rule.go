package types

import (
	"time"
)

type AgedRule struct {
	RuleSource
	HourMultiplier float64 `json:"hourMultiplier"`
}

func (_ *AgedRule) Type() RuleType {
	return "AgedRule"
}

func (_ *AgedRule) New() JsonType {
	return &AgedRule{}
}

func (r *AgedRule) GetValue(item Item) float64 {

	value := r.GetSourceValue(item)

	now := time.Now().UnixMilli()
	v, ok := AsNumber[int64](value)
	if ok && v > 0 {
		return float64((now-v)/60_000) * r.HourMultiplier
	} else {
		return 0
	}
}
