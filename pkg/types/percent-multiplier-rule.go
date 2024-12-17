package types

import (
	"sync"
)

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

func (r *PercentMultiplierRule) GetValue(item Item, res chan<- float64, wg *sync.WaitGroup) {
	defer wg.Done()
	value := r.GetSourceValue(item)
	v, ok := AsNumber[float64](value)

	if ok && v >= r.Max && v <= r.Min {
		res <- v * r.Multiplier
	} else {
		res <- 0
	}
}
