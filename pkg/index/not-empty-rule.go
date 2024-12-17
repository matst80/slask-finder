package index

import (
	"github.com/matst80/slask-finder/pkg/types"
	"sync"
)

type NotEmptyRule struct {
	RuleSource
	ValueIfMatch    float64 `json:"value,omitempty"`
	ValueIfNotMatch float64 `json:"valueIfNotMatch"`
}

func (_ *NotEmptyRule) Type() RuleType {
	return "NotEmptyRule"
}

func (_ *NotEmptyRule) New() ItemPopularityRule {
	return &NotEmptyRule{}
}

func (r *NotEmptyRule) GetValue(item types.Item, res chan<- float64, wg *sync.WaitGroup) {
	defer wg.Done()
	value := r.GetSourceValue(item)
	match := false
	if value == nil {
		res <- r.ValueIfNotMatch
		return
	}
	switch v := value.(type) {
	case string:
		if v != "" {
			match = true
		}
	case float64:
		if v != 0 {
			match = true
		}
	case int:
		if v > 0 {
			match = true
		}
	}
	if match {
		res <- r.ValueIfMatch
	} else {
		res <- r.ValueIfNotMatch
	}
}
