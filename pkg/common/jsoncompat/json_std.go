//go:build !jsonv2

package jsoncompat

import "encoding/json"

// Marshal proxies to the standard library json.Marshal when jsonv2 build tag is absent.
func Marshal(v any) ([]byte, error) { return json.Marshal(v) }

// Unmarshal proxies to the standard library json.Unmarshal when jsonv2 build tag is absent.
func Unmarshal(data []byte, v any) error { return json.Unmarshal(data, v) }
