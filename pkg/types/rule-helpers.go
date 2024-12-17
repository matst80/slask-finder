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

		b.Write(slices.Insert(item, len(item)-1, []byte(fmt.Sprintf(`,"$type":"%s"`, i.Type()))...))
		if idx < len(l)-1 {
			b.Write([]byte(","))
		}
	}
	b.Write([]byte("]"))
	return b.Bytes(), nil
}

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

func FromJsonTypes[K interface{}](arr JsonTypes) []K {
	var result []K
	for _, v := range arr {
		r, ok := v.(K)
		if !ok {
			continue
		}
		result = append(result, r)
	}
	return result
}
