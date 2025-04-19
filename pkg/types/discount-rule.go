package types

type DiscountRule struct {
	Multiplier   float64 `json:"multiplier"`
	ValueIfMatch float64 `json:"valueIfMatch"`
}

func (r *DiscountRule) Type() RuleType {
	return "DiscountRule"
}

func (r *DiscountRule) New() JsonType {
	return &DiscountRule{}
}

func (r *DiscountRule) GetValue(item Item) float64 {

	price := float64(item.GetPrice())
	discountP := item.GetDiscount()

	if discountP > 0 {
		discount := float64(discountP)
		p := discount / price
		return r.ValueIfMatch + (p * r.Multiplier)
	}
	return 0
}
