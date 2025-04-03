package types

import (
	"testing"
)

func TestBinaryEncoding(t *testing.T) {
	// Test the binary encoding and decoding of ItemFields
	fields := ItemFields{}

	fields[1] = "stringValue"
	fields[2] = 123
	fields[3] = []string{"value1", "value2", "value3", ""}
	fields[4] = []string{}
	fields[5] = []string{""}

	encoded, err := fields.MarshalBinary()
	if err != nil {
		t.Fatalf("Failed to encode ItemFields: %v", err)
	}

	var decoded ItemFields
	err = decoded.UnmarshalBinary(encoded)
	if err != nil {
		t.Fatalf("Failed to decode ItemFields: %v", err)
	}

	if len(decoded) != len(fields) {
		t.Fatalf("Decoded fields length mismatch: got %d, want %d", len(decoded), len(fields))
	}

	for k, v := range fields {
		decValue, ok := decoded[k]
		if !ok {
			t.Errorf("Missing key %d in decoded fields", k)
			continue
		}
		switch val := v.(type) {
		case string:
			if decValue != val {
				t.Errorf("Decoded string field mismatch for key %d: got %s, want %s", k, decoded[k], v)
			}
		case int:
			if decValue != val {
				t.Errorf("Decoded number field mismatch for key %d: got %s, want %s", k, decoded[k], v)
			}
		case []string:
			if len(decValue.([]string)) != len(val) {
				t.Errorf("Decoded array length mismatch for key %d: got %d, want %d", k, len(decoded[k].([]string)), len(val))
			}
			for i, part := range val {
				if decValue.([]string)[i] != part {
					t.Errorf("Decoded array value mismatch for key %d at index %d: got %s, want %s", k, i, decoded[k].([]string)[i], v)
				}
			}
		default:
			t.Errorf("Unexpected type for key %d: got %T", k, val)
		}
	}
}
