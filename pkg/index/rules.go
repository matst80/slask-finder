package index

import (
	"github.com/matst80/slask-finder/pkg/types"
	"reflect"
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

type ItemPopularityRule interface {
	GetValue(item types.Item, res chan<- float64, wg *sync.WaitGroup)
}

type NotEmptyRule struct {
	PropertyName    string  `json:"property,omitempty"`
	FieldId         uint    `json:"fieldId,omitempty"`
	ValueIfMatch    float64 `json:"value,omitempty"`
	ValueIfNotMatch float64 `json:"valueIfNotMatch"`
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
	NoStoreMultiplier float64
	NoStockValue      float64
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
	Multiplier   float64
	ValueIfMatch float64
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
	Multiplier     float64
	SubtractValue  int
	ValueIfNoMatch float64
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
