package types

import (
	"sync"
)

type ItemPopularityRule interface {
	GetValue(item Item, res chan<- float64, wg *sync.WaitGroup)
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
	wg := &sync.WaitGroup{}
	res := make(chan float64)
	for _, rule := range rules {
		wg.Add(1)
		go rule.GetValue(item, res, wg)
	}
	go func() {
		wg.Wait()
		defer close(res)
	}()

	var sum float64
	for v := range res {
		sum += v
	}
	return sum
}
