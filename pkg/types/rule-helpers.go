package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
)

type RuleType string

type LazyType struct {
	Type RuleType `json:"$type"`
}

type JsonType interface {
	Type() RuleType
	New() JsonType
}

var lookup = make(map[RuleType]JsonType)

func Register(iface JsonType) {
	lookup[iface.Type()] = iface
}

type JsonTypes []JsonType

func (l *JsonTypes) UnmarshalJSON(b []byte) error {
	var raw []json.RawMessage
	err := json.Unmarshal(b, &raw)
	if err != nil {
		return err
	}
	// Allocate an array of MyInterface
	*l = make(JsonTypes, len(raw))
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

func (l JsonTypes) MarshalJSON() ([]byte, error) {
	b := bytes.Buffer{}
	b.Write([]byte("["))
	for idx, i := range l {
		item, err := json.Marshal(i)
		if err != nil {
			continue
		}

		b.Write(slices.Insert(item, len(item)-1, fmt.Appendf(nil, `,"$type":"%s"`, i.Type())...))
		if idx < len(l)-1 {
			b.Write([]byte(","))
		}
	}
	b.Write([]byte("]"))
	return b.Bytes(), nil
}

func GetPropertyValue(item any, propertyName string) any {
	val := reflect.ValueOf(item).Elem()

	field := val.FieldByName(propertyName)
	if !field.IsValid() {
		return nil
	}
	return field.Interface()
}

func AsNumber[K int | float64 | int64](value any) (K, bool) {
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

func FromJsonTypes[K any](arr JsonTypes) []K {
	// Preallocate with full length; capacity hint avoids reallocations.
	// We still skip elements that don't assert to K; final len may be < cap.
	result := make([]K, 0, len(arr))
	for _, v := range arr {
		if r, ok := v.(K); ok {
			result = append(result, r)
		}
	}
	return result
}
