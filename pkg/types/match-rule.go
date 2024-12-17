package types

import (
	"sync"
)

type MatchRule struct {
	RuleSource
	Match           interface{} `json:"match"`
	Invert          bool        `json:"invert,omitempty"`
	ValueIfMatch    float64     `json:"value"`
	ValueIfNotMatch float64     `json:"valueIfNotMatch"`
}

func (_ *MatchRule) Type() RuleType {
	return "MatchRule"
}

func (_ *MatchRule) New() ItemPopularityRule {
	return &MatchRule{}
}

func (r *MatchRule) GetValue(item Item, res chan<- float64, wg *sync.WaitGroup) {
	defer wg.Done()
	match := false
	value := r.GetSourceValue(item)

	if r.Invert {
		match = value != r.Match
	} else {
		match = value == r.Match
	}

	if match {
		res <- r.ValueIfMatch
	} else {
		res <- r.ValueIfNotMatch
	}
}
