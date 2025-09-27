//go:build jsonv2

package types

import (
	json "encoding/json/v2"
	"reflect"
	"sort"
	"testing"
)

// helper to create a deterministic sample ItemFields
func sampleItemFields() ItemFields {
	return ItemFields{
		3:  5,
		4:  9999,
		5:  12999,
		6:  4,
		7:  25,
		8:  "Some bullet points here",
		9:  "Elgiganten",
		10: "Normal",
		// Expect slice as []interface{} to align with decoder default
		11: []interface{}{"tag1", "tag2"},
		12: 1.23,
		13: true,
		14: map[string]any{"nested": "obj", "arr": []interface{}{float64(1), float64(2), float64(3)}},
	}
}

// collect sorted ids for comparison logging
func sortedKeys(m ItemFields) []uint {
	keys := make([]uint, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

func TestItemFields_Compact_RoundTrip(t *testing.T) {
	EmitCompactItemFields = true
	DeterministicItemFields = true
	original := sampleItemFields()

	b, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal compact failed: %v", err)
	}

	var decoded ItemFields
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal compact failed: %v", err)
	}

	if len(decoded) != len(original) {
		t.Fatalf("length mismatch: got %d want %d", len(decoded), len(original))
	}

	for k, v := range original {
		dv, ok := decoded[k]
		if !ok {
			t.Fatalf("missing key %d", k)
		}
		if !reflect.DeepEqual(dv, v) {
			t.Fatalf("value mismatch for key %d: got %#v want %#v", k, dv, v)
		}
	}
}

func TestItemFields_Object_StillDecodes_WhenCompactEnabled(t *testing.T) {
	// ensure we can still read legacy object JSON even if emitting compact now
	EmitCompactItemFields = true
	DeterministicItemFields = true
	objJSON := []byte(`{"3":5,"4":9999,"5":12999,"6":4,"7":25,"8":"Some bullet points here","9":"Elgiganten","10":"Normal","11":["tag1","tag2"],"12":1.23,"13":true,"14":{"nested":"obj","arr":[1,2,3]}}`)

	var decoded ItemFields
	if err := json.Unmarshal(objJSON, &decoded); err != nil {
		t.Fatalf("unmarshal object form under compact failed: %v", err)
	}

	expected := sampleItemFields()
	if len(decoded) != len(expected) {
		t.Fatalf("len mismatch got %d want %d", len(decoded), len(expected))
	}

	for _, k := range sortedKeys(expected) {
		if !reflect.DeepEqual(decoded[k], expected[k]) {
			t.Fatalf("mismatch key %d got %#v want %#v", k, decoded[k], expected[k])
		}
	}
}

func TestItemFields_Object_Vs_Compact_Equivalence(t *testing.T) {
	DeterministicItemFields = true
	orig := sampleItemFields()

	// Marshal object form
	EmitCompactItemFields = false
	objBytes, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal object failed: %v", err)
	}

	// Marshal compact form
	EmitCompactItemFields = true
	cmpBytes, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal compact failed: %v", err)
	}

	// Both must decode to equivalent maps
	var objDecoded, cmpDecoded ItemFields
	if err := json.Unmarshal(objBytes, &objDecoded); err != nil {
		t.Fatalf("unmarshal object bytes failed: %v", err)
	}
	if err := json.Unmarshal(cmpBytes, &cmpDecoded); err != nil {
		t.Fatalf("unmarshal compact bytes failed: %v", err)
	}

	if len(objDecoded) != len(cmpDecoded) {
		t.Fatalf("decoded length mismatch: %d vs %d", len(objDecoded), len(cmpDecoded))
	}
	for k, v := range objDecoded {
		cv, ok := cmpDecoded[k]
		if !ok {
			t.Fatalf("missing key %d in compact", k)
		}
		if !reflect.DeepEqual(cv, v) {
			t.Fatalf("value mismatch key %d: object=%#v compact=%#v", k, v, cv)
		}
	}

	if len(cmpBytes) >= len(objBytes) {
		t.Logf("NOTE: compact bytes (%d) >= object bytes (%d) for this small sample; benefit shows on larger field counts", len(cmpBytes), len(objBytes))
	} else {
		t.Logf("Compact saved %d bytes", len(objBytes)-len(cmpBytes))
	}
}
