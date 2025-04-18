package index

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEncodingFlag(t *testing.T) {
	AllowConditionalData = false
	// Create a sample DataItem
	item := ItemProp{

		MarginPercent: 4.0,
	}

	bytes, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("Failed to marshal item: %v", err)
	}
	str := string(bytes)
	if strings.Contains(str, `"mp":4`) {
		t.Fatalf("Expected JSON to be %s, got %s", `{"mp":4}`, str)
	}
}

func TestEncodingFlagAllowed(t *testing.T) {
	AllowConditionalData = true
	// Create a sample DataItem
	item := ItemProp{

		MarginPercent: 4.0,
	}

	bytes, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("Failed to marshal item: %v", err)
	}
	str := string(bytes)
	if !strings.Contains(str, `"mp":4`) {
		t.Fatalf("Expected JSON to be %s, got %s", `{"mp":4}`, str)
	}
}
