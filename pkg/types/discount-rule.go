package types

import (
	"sync"
)

type DiscountRule struct {
	Multiplier   float64 `json:"multiplier"`
	ValueIfMatch float64 `json:"valueIfMatch"`
}

func (_ *DiscountRule) Type() RuleType {
	return "DiscountRule"
}

func (_ *DiscountRule) New() JsonType {
	return &DiscountRule{}
}

func (r *DiscountRule) GetValue(item Item, res chan<- float64, wg *sync.WaitGroup) {
	defer wg.Done()
	price := float64(item.GetPrice())
	discountP := item.GetDiscount()

	if discountP == nil {
		res <- 0
	} else if *discountP > 0 {
		discount := float64(*discountP)
		p := discount / price
		res <- r.ValueIfMatch + p*r.Multiplier
	} else {
		res <- 0
	}
}
