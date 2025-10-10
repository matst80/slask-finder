package main

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRoundTrip(t *testing.T) {
	orig := MustStoreDate(2025, 12, 24)
	b, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != "[2025,12,24]" {
		t.Fatalf("unexpected json: %s", b)
	}
	var got StoreDate
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !orig.Equal(got) {
		t.Fatalf("mismatch: orig=%v got=%v", orig, got)
	}
}

func TestInvalidLengths(t *testing.T) {
	var sd StoreDate
	if err := json.Unmarshal([]byte("[2023,1]"), &sd); err == nil {
		t.Fatal("expected error for short array")
	}
	if err := json.Unmarshal([]byte("[2023,1,2,3]"), &sd); err == nil {
		t.Fatal("expected error for long array")
	}
}

func TestValidation(t *testing.T) {
	_, err := NewStoreDate(0, 1, 1)
	if err == nil {
		t.Fatal("expected year out of range")
	}
	_, err = NewStoreDate(2024, 13, 1)
	if err == nil {
		t.Fatal("expected month out of range")
	}
	_, err = NewStoreDate(2023, 2, 29) // not leap year
	if err == nil {
		t.Fatal("expected day out of range")
	}
	_, err = NewStoreDate(2024, 2, 29) // leap year
	if err != nil {
		t.Fatalf("unexpected leap year error: %v", err)
	}
}

func TestISO(t *testing.T) {
	sd := MustStoreDate(2025, 12, 24)
	if sd.ISO() != "2025-12-24" {
		t.Fatalf("iso mismatch: %s", sd.ISO())
	}
	if sd.String() != "2025-12-24" {
		t.Fatalf("string mismatch: %s", sd.String())
	}
}

func TestParseISO(t *testing.T) {
	sd, err := ParseISO("2025-12-24")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if sd.Year != 2025 || sd.Month != 12 || sd.Day != 24 {
		t.Fatalf("unexpected parsed date: %+v", sd)
	}
}

func TestFromTime(t *testing.T) {
	now := time.Date(2030, 6, 2, 15, 4, 5, 0, time.UTC)
	sd := FromTime(now)
	if sd.Year != 2030 || sd.Month != 6 || sd.Day != 2 {
		t.Fatalf("unexpected from time: %+v", sd)
	}
}

func TestOrdering(t *testing.T) {
	a := MustStoreDate(2025, 12, 24)
	b := MustStoreDate(2025, 12, 25)
	if !a.Before(b) {
		t.Fatal("expected a before b")
	}
	if !b.After(a) {
		t.Fatal("expected b after a")
	}
	if a.After(b) || b.Before(a) {
		t.Fatal("unexpected ordering")
	}
	if !a.Equal(a) {
		t.Fatal("expected equality")
	}
}

func TestAddDays(t *testing.T) {
	start := MustStoreDate(2025, 12, 24)
	plus := start.AddDays(7)
	if plus.ISO() != "2025-12-31" {
		t.Fatalf("expected 2025-12-31 got %s", plus.ISO())
	}
	next := plus.AddDays(1)
	if next.ISO() != "2026-01-01" {
		t.Fatalf("rollover failed got %s", next.ISO())
	}
	back := next.AddDays(-8)
	if !back.Equal(start) {
		t.Fatalf("expected back to original, got %s", back.ISO())
	}
}

func TestDaysSince(t *testing.T) {
	a := MustStoreDate(2025, 12, 24)
	b := MustStoreDate(2025, 12, 31)
	if d := b.DaysSince(a); d != 7 {
		t.Fatalf("expected 7 got %d", d)
	}
	if d := a.DaysSince(b); d != -7 {
		t.Fatalf("expected -7 got %d", d)
	}
}

func TestFlexibleParsing(t *testing.T) {
	sd, err := ParseFlexible("2025-12-24T10:11:12Z")
	if err != nil {
		t.Fatalf("flex parse failed: %v", err)
	}
	if sd.ISO() != "2025-12-24" {
		t.Fatalf("unexpected flex iso: %s", sd.ISO())
	}
	sd2, err := ParseFlexible("2025-12-24")
	if err != nil || sd2.ISO() != "2025-12-24" {
		t.Fatalf("unexpected flex iso2: %v %s", err, sd2.ISO())
	}
}

func TestJSONErrorMessages(t *testing.T) {
	var sd StoreDate
	if err := json.Unmarshal([]byte(`"2025-12-24"`), &sd); err == nil {
		t.Fatal("expected error for non-array json")
	}
}
