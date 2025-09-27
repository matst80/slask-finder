//go:build jsonv2

package types

import (
	json "encoding/json/v2"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

// generate random ItemFields with a mix of value shapes
func randomItemFields(r *rand.Rand) ItemFields {
	n := r.Intn(20) // up to 20 fields
	f := make(ItemFields, n)
	for i := 0; i < n; i++ {
		id := uint(r.Intn(200)) // small id space to increase overwrite collisions
		switch r.Intn(7) {
		case 0:
			f[id] = r.Intn(100000)
		case 1:
			f[id] = r.Float64()*1e6 - 5e5
		case 2:
			f[id] = (r.Intn(2) == 0)
		case 3:
			// short string
			ln := r.Intn(8) + 1
			b := make([]rune, ln)
			for i := 0; i < ln; i++ {
				b[i] = rune('a' + r.Intn(26))
			}
			f[id] = string(b)
		case 4:
			// slice of small strings as []interface{}
			ln := r.Intn(5)
			arr := make([]interface{}, ln)
			for i := 0; i < ln; i++ {
				arr[i] = string(rune('a' + r.Intn(26)))
			}
			f[id] = arr
		case 5:
			// nested object
			m := map[string]interface{}{"k": r.Intn(10)}
			if r.Intn(2) == 0 {
				m["s"] = "str"
			}
			f[id] = m
		case 6:
			// nested mixed array
			ln := r.Intn(4)
			arr := make([]interface{}, ln)
			for i := 0; i < ln; i++ {
				switch r.Intn(3) {
				case 0:
					arr[i] = r.Intn(50)
				case 1:
					arr[i] = string(rune('a' + r.Intn(26)))
				case 2:
					arr[i] = (r.Intn(2) == 0)
				}
			}
			f[id] = arr
		}
	}
	return f
}

// FuzzItemFields ensures object vs compact encodings round-trip to equivalent maps.
func FuzzItemFields(f *testing.F) {
	// seed with a couple deterministic cases
	f.Add([]byte(`{"1":1,"2":"a"}`))
	f.Add([]byte(`[]`)) // also accept compact empty (will decode as empty map)

	f.Fuzz(func(t *testing.T, data []byte) {
		// Try to decode given input into ItemFields (object or compact form)
		EmitCompactItemFields = false
		var base ItemFields
		if err := json.Unmarshal(data, &base); err != nil {
			// Not valid ItemFields; ignore
			t.Skip()
		}

		// Now create a randomized augmentation merged with base to exercise encoder
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		randExtra := randomItemFields(r)
		for k, v := range randExtra {
			base[k] = v
		}

		// Encode object form
		EmitCompactItemFields = false
		objBytes, err := json.Marshal(base)
		if err != nil {
			t.Fatalf("marshal object: %v", err)
		}

		// Encode compact form
		EmitCompactItemFields = true
		cmpBytes, err := json.Marshal(base)
		if err != nil {
			t.Fatalf("marshal compact: %v", err)
		}

		// Decode both
		var objDecoded, cmpDecoded ItemFields
		if err := json.Unmarshal(objBytes, &objDecoded); err != nil {
			t.Fatalf("unmarshal obj back: %v", err)
		}
		if err := json.Unmarshal(cmpBytes, &cmpDecoded); err != nil {
			t.Fatalf("unmarshal cmp back: %v", err)
		}

		if len(objDecoded) != len(cmpDecoded) {
			t.Fatalf("len mismatch obj=%d cmp=%d", len(objDecoded), len(cmpDecoded))
		}
		for k, v := range objDecoded {
			cv, ok := cmpDecoded[k]
			if !ok {
				t.Fatalf("missing key %d in compact", k)
			}
			if !reflect.DeepEqual(v, cv) {
				t.Fatalf("value mismatch key %d: %#v vs %#v", k, v, cv)
			}
		}
	})
}
