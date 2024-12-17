package types

import (
	"sync"
	"time"
)

type AgedRule struct {
	RuleSource
	HourMultiplier float64 `json:"hourMultiplier"`
}

func (_ *AgedRule) Type() RuleType {
	return "AgedRule"
}

func (_ *AgedRule) New() ItemPopularityRule {
	return &AgedRule{}
}

func (r *AgedRule) GetValue(item Item, res chan<- float64, wg *sync.WaitGroup) {
	defer wg.Done()
	value := r.GetSourceValue(item)

	now := time.Now().UnixNano()
	v, ok := AsNumber[int64](value)
	if ok {
		res <- float64((now-v)/60_000) * r.HourMultiplier
	} else {
		res <- 0
	}
}
