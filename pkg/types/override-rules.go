package types

import (
	"sync"
)

type FilterOverrideules []FilterOverrideRule

type FilterOverrideRule interface {
	GetValue(item Item, res chan<- float64, wg *sync.WaitGroup)
}

func init() {
	//RegisterRule(&MatchRule{})
}

func MatchOverrides(item Item, rules ...FilterOverrideRule) float64 {
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
