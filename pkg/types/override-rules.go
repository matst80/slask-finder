package types

import (
	"sync"
)

type FilterOverrideules []FilterMatchRule

type RuleActionType string

type RuleAction struct {
	Value interface{}    `json:"value"`
	Type  RuleActionType `json:"id"`
}

const (
	Audience RuleActionType = "audience"
	Ignore   RuleActionType = "ignore"
)

type FilterMatchRule interface {
	GetValue(item FacetRequest, res chan<- *RuleAction, wg *sync.WaitGroup)
}

func init() {
	//RegisterRule(&MatchRule{})
}

func MatchOverrides(item FacetRequest, rules ...FilterMatchRule) *RuleAction {
	wg := &sync.WaitGroup{}
	res := make(chan *RuleAction)
	for _, rule := range rules {
		wg.Add(1)
		go rule.GetValue(item, res, wg)
	}
	go func() {
		wg.Wait()
		defer close(res)
	}()

	for v := range res {
		if v != nil {
			return v
		}
	}
	return nil
}
