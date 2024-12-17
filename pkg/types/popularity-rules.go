package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"sync"
)

func GetPropertyValue(item interface{}, propertyName string) interface{} {
	val := reflect.ValueOf(item).Elem()
	field := val.FieldByName(propertyName)
	if !field.IsValid() {
		return nil
	}
	return field.Interface()
}

func AsNumber[K int | float64 | int64](value interface{}) (K, bool) {
	found := false
	var v K
	switch input := value.(type) {
	case int:
		v = K(input)
		found = true
	case float64:
		v = K(input)
		found = true
	case int64:
		v = K(input)
		found = true
	}
	return v, found
}

type RuleType string
type ItemPopularityRule interface {
	Type() RuleType
	New() ItemPopularityRule
	GetValue(item Item, res chan<- float64, wg *sync.WaitGroup)
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
