package index

import (
	"github.com/matst80/slask-finder/pkg/types"
	"sync"
)

type RatingRule struct {
	Multiplier     float64 `json:"multiplier,omitempty"`
	SubtractValue  int     `json:"subtractValue,omitempty"`
	ValueIfNoMatch float64 `json:"valueIfNoMatch,omitempty"`
}

func (_ *RatingRule) Type() RuleType {
	return "RatingRule"
}

func (_ *RatingRule) New() ItemPopularityRule {
	return &RatingRule{}
}

func (r *RatingRule) GetValue(item types.Item, res chan<- float64, wg *sync.WaitGroup) {
	defer wg.Done()
	avg, num := item.GetRating()
	if r.Multiplier == 0 {
		r.Multiplier = 1
	}
	if num == 0 {
		res <- r.ValueIfNoMatch
	} else {
		res <- (float64(avg) - float64(r.SubtractValue)) * r.Multiplier
	}
}
