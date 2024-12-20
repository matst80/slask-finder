package types

type ItemPopularityRule interface {
	GetValue(item Item) float64
}

func init() {
	Register(&MatchRule{})
	Register(&DiscountRule{})
	Register(&OutOfStockRule{})
	Register(&NumberLimitRule{})
	Register(&PercentMultiplierRule{})
	Register(&RatingRule{})
	Register(&AgedRule{})
}

type ItemPopularityRules []ItemPopularityRule

func CollectPopularity(item Item, rules ...ItemPopularityRule) float64 {
	var sum float64
	for _, rule := range rules {
		sum += rule.GetValue(item)
	}

	return sum
}
