package index

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/matst80/slask-finder/pkg/types"
	"reflect"
	"slices"
	"sync"
	"time"
)

func GetPropertyValue(item interface{}, propertyName string) interface{} {
	val := reflect.ValueOf(item).Elem()
	field := val.FieldByName(propertyName)
	if !field.IsValid() {
		return nil
	}
	return field.Interface()
}

type RuleType string
type ItemPopularityRule interface {
	Type() RuleType
	New() ItemPopularityRule
	GetValue(item types.Item, res chan<- float64, wg *sync.WaitGroup)
}

type LazyType struct {
	Type RuleType `json:"$type"`
}

var lookup = make(map[RuleType]ItemPopularityRule)

func RegisterRule(iface ItemPopularityRule) {
	lookup[iface.Type()] = iface
}

func init() {
	RegisterRule(&MatchRule{})
	RegisterRule(&DiscountRule{})
	RegisterRule(&OutOfStockRule{})
	RegisterRule(&NumberLimitRule{})
	RegisterRule(&PercentMultiplierRule{})
	RegisterRule(&RatingRule{})
	RegisterRule(&AgedRule{})
}

func (l *ItemPopularityRules) UnmarshalJSON(b []byte) error {
	var raw []json.RawMessage
	err := json.Unmarshal(b, &raw)
	if err != nil {
		return err
	}
	// Allocate an array of MyInterface
	*l = make(ItemPopularityRules, len(raw))
	var t LazyType
	for i, r := range raw {
		// Unmarshal the array first into a type array
		err := json.Unmarshal(r, &t)
		if err != nil {
			return err
		}
		// Create an instance of the type
		myInterfaceFunc, ok := lookup[t.Type]
		if !ok {
			return fmt.Errorf("unregistered interface type : %s", t.Type)
		}
		myInterface := myInterfaceFunc.New()
		err = json.Unmarshal(r, myInterface)
		if err != nil {
			return err
		}
		(*l)[i] = myInterface
	}
	return nil
}

func (l ItemPopularityRules) MarshalJSON() ([]byte, error) {
	b := bytes.Buffer{}
	b.Write([]byte("["))
	for idx, i := range l {
		item, err := json.Marshal(i)
		if err != nil {
			continue
		}

		b.Write(slices.Insert(item, len(item)-1, []byte(fmt.Sprintf(`,"$type":"%s"`, i.Type()))...))
		if idx < len(l)-1 {
			b.Write([]byte(","))
		}
	}
	b.Write([]byte("]"))
	return b.Bytes(), nil
}

type ItemPopularityRules []ItemPopularityRule

type NotEmptyRule struct {
	PropertyName    string  `json:"property,omitempty"`
	FieldId         uint    `json:"fieldId,omitempty"`
	ValueIfMatch    float64 `json:"value,omitempty"`
	ValueIfNotMatch float64 `json:"valueIfNotMatch"`
}

func (_ NotEmptyRule) Type() RuleType {
	return "NotEmptyRule"
}

func (_ NotEmptyRule) New() ItemPopularityRule {
	return &NotEmptyRule{}
}

func (r *NotEmptyRule) GetValue(item types.Item, res chan<- float64, wg *sync.WaitGroup) {
	defer wg.Done()
	var value interface{}
	match := false
	if r.FieldId > 0 {
		value = item.GetFields()[r.FieldId]
	} else if r.PropertyName != "" {
		value = GetPropertyValue(item, r.PropertyName)
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
		res <- 0
	}
}

type MatchRule struct {
	Match           interface{} `json:"match"`
	Invert          bool        `json:"invert,omitempty"`
	PropertyName    string      `json:"property,omitempty"`
	FieldId         uint        `json:"fieldId,omitempty"`
	ValueIfMatch    float64     `json:"value"`
	ValueIfNotMatch float64     `json:"valueIfNotMatch"`
}

func (_ *MatchRule) Type() RuleType {
	return "MatchRule"
}

func (_ *MatchRule) New() ItemPopularityRule {
	return &MatchRule{}
}

func (r *MatchRule) GetValue(item types.Item, res chan<- float64, wg *sync.WaitGroup) {
	defer wg.Done()
	var value interface{}
	match := false
	if r.FieldId > 0 {
		value = item.GetFields()[r.FieldId]
	} else if r.PropertyName != "" {
		value = GetPropertyValue(item, r.PropertyName)
	}
	if r.Invert {
		match = value != r.Match
	} else {
		match = value == r.Match
	}
	//switch v := value.(type) {
	//case string:
	//
	//}
	if match {
		res <- r.ValueIfMatch
	} else {
		res <- r.ValueIfNotMatch
	}
}

func CollectPopularity(item types.Item, rules ...ItemPopularityRule) float64 {
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

type OutOfStockRule struct {
	NoStoreMultiplier float64 `json:"noStoreMultiplier"`
	NoStockValue      float64 `json:"noStockValue"`
}

func (_ *OutOfStockRule) Type() RuleType {
	return "MatchRule"
}

func (_ *OutOfStockRule) New() ItemPopularityRule {
	return &OutOfStockRule{}
}

func (r *OutOfStockRule) GetValue(item types.Item, res chan<- float64, wg *sync.WaitGroup) {
	defer wg.Done()
	stores := len(item.GetStock())
	if stores > 0 {
		res <- float64(stores) * r.NoStoreMultiplier
		return
	}
	level := GetPropertyValue(item, "StockLevel")
	hasStock := false
	switch l := level.(type) {
	case string:
		hasStock = l != "" && l != "0"
	}
	if hasStock {
		res <- 0
	} else {
		res <- r.NoStockValue
	}
}

type DiscountRule struct {
	Multiplier   float64 `json:"multiplier"`
	ValueIfMatch float64 `json:"valueIfMatch"`
}

func (_ *DiscountRule) Type() RuleType {
	return "DiscountRule"
}

func (_ *DiscountRule) New() ItemPopularityRule {
	return &DiscountRule{}
}

func (r *DiscountRule) GetValue(item types.Item, res chan<- float64, wg *sync.WaitGroup) {
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

type RatingRule struct {
	Multiplier     float64 `json:"multiplier"`
	SubtractValue  int     `json:"subtractValue"`
	ValueIfNoMatch float64 `json:"valueIfNoMatch"`
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
	if num == 0 {
		res <- r.ValueIfNoMatch
	} else {
		res <- (float64(avg) - float64(r.SubtractValue)) * r.Multiplier
	}
}

type NumberComparator string

const (
	Over  NumberComparator = ">"
	Under NumberComparator = "<"
	Equal NumberComparator = "="
)

type NumberLimitRule struct {
	Limit           float64          `json:"limit"`
	Comparator      NumberComparator `json:"comparator"`
	ValueIfMatch    float64          `json:"value"`
	ValueIfNotMatch float64          `json:"valueIfNotMatch"`
	PropertyName    string           `json:"property,omitempty"`
	FieldId         uint             `json:"fieldId,omitempty"`
}

func (_ *NumberLimitRule) Type() RuleType {
	return "NumberLimitRule"
}

func (_ *NumberLimitRule) New() ItemPopularityRule {
	return &NumberLimitRule{}
}

func (r *NumberLimitRule) GetValue(item types.Item, res chan<- float64, wg *sync.WaitGroup) {
	defer wg.Done()
	var value interface{}
	found := false
	matchFn := func(nr float64) bool {
		switch r.Comparator {
		case Over:
			return nr > r.Limit
		case Under:
			return nr < r.Limit
		case Equal:
			return nr == r.Limit
		}
		return false
	}
	if r.FieldId > 0 {
		value = item.GetFields()[r.FieldId]
	} else if r.PropertyName != "" {
		value = GetPropertyValue(item, r.PropertyName)
	}
	v := 0.0
	switch input := value.(type) {
	case int:
		v = float64(input)
		found = true
	case float64:
		v = input
		found = true
	case int64:
		v = float64(input)
		found = true
	}
	if !found {
		res <- r.ValueIfNotMatch
	} else {
		if matchFn(v) {
			res <- r.ValueIfMatch
		} else {
			res <- r.ValueIfNotMatch
		}
	}
}

type PercentMultiplierRule struct {
	Multiplier   float64 `json:"multiplier"`
	Min          float64 `json:"min"`
	Max          float64 `json:"max"`
	PropertyName string  `json:"property,omitempty"`
	FieldId      uint    `json:"fieldId,omitempty"`
}

func (_ *PercentMultiplierRule) Type() RuleType {
	return "NumberLimitRule"
}

func (_ *PercentMultiplierRule) New() ItemPopularityRule {
	return &PercentMultiplierRule{}
}

func (r *PercentMultiplierRule) GetValue(item types.Item, res chan<- float64, wg *sync.WaitGroup) {
	defer wg.Done()
	var value = interface{}(0)
	if r.FieldId > 0 {
		value = item.GetFields()[r.FieldId]
	} else if r.PropertyName != "" {
		value = GetPropertyValue(item, r.PropertyName)
	}
	switch v := value.(type) {
	case int:
		value = float64(v)
	case float64:
		value = v
	case uint:
		value = float64(v)
	}
	if v, ok := value.(float64); ok {
		if v < r.Min {
			res <- 0
		} else if v > r.Max {
			res <- 0
		} else {
			res <- v * r.Multiplier
		}
	} else {
	}
}

type AgedRule struct {
	HourMultiplier float64 `json:"hourMultiplier"`
	PropertyName   string  `json:"property,omitempty"`
	FieldId        uint    `json:"fieldId,omitempty"`
}

func (_ *AgedRule) Type() RuleType {
	return "AgedRule"
}

func (_ *AgedRule) New() ItemPopularityRule {
	return &AgedRule{}
}

func (r *AgedRule) GetValue(item types.Item, res chan<- float64, wg *sync.WaitGroup) {
	defer wg.Done()
	var value interface{}
	if r.FieldId > 0 {
		value = item.GetFields()[r.FieldId]
	} else if r.PropertyName != "" {
		value = GetPropertyValue(item, r.PropertyName)
	}
	now := time.Now().UnixNano()
	switch v := value.(type) {
	case int:
		res <- float64((now-int64(v))/60_000) * r.HourMultiplier
	case int64:
		res <- float64((now-v)/60_000) * r.HourMultiplier
	}
}
