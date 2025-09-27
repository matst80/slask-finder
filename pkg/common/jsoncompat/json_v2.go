//go:build jsonv2

package jsoncompat

import json "encoding/json/v2"

// Marshal proxies to encoding/json/v2 Marshal when jsonv2 build tag is present.
func Marshal(v any) ([]byte, error) { return json.Marshal(v) }

// Unmarshal proxies to encoding/json/v2 Unmarshal when jsonv2 build tag is present.
func Unmarshal(data []byte, v any) error { return json.Unmarshal(data, v) }
