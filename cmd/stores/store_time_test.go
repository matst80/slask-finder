package main

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestStoreTimeRoundTrip(t *testing.T) {
	orig := MustStoreTime(13, 45, 7)
	b, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != "[13,45,7]" {
		t.Fatalf("unexpected json: %s", b)
	}

	var got StoreTime
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !orig.Equal(got) {
		t.Fatalf("mismatch: orig=%v got=%v", orig, got)
	}
}

func TestStoreTimeMarshalInvalid(t *testing.T) {
	invalid := StoreTime{Hour: 25, Minute: 0, Second: 0}
	if _, err := json.Marshal(invalid); err == nil {
		t.Fatal("expected error for invalid hour")
	}
}

func TestStoreTimeUnmarshalErrors(t *testing.T) {
	var st StoreTime

	// Not an array
	if err := json.Unmarshal([]byte(`"13:00:00"`), &st); err == nil {
		t.Fatal("expected error for string form")
	}

	// Too few elements
	if err := json.Unmarshal([]byte(`[13,0]`), &st); err == nil {
		t.Fatal("expected error for short array")
	}

	// Too many elements
	if err := json.Unmarshal([]byte(`[13,0,0,5]`), &st); err == nil {
		t.Fatal("expected error for long array")
	}

	// Out of range values
	if err := json.Unmarshal([]byte(`[24,0,0]`), &st); err == nil {
		t.Fatal("expected error for hour 24")
	}
	if err := json.Unmarshal([]byte(`[23,60,0]`), &st); err == nil {
		t.Fatal("expected error for minute 60")
	}
	if err := json.Unmarshal([]byte(`[23,59,60]`), &st); err == nil {
		t.Fatal("expected error for second 60")
	}
}

func TestStoreTimeValidate(t *testing.T) {
	_, err := NewStoreTime(-1, 0, 0)
	if !errors.Is(err, ErrHourOutOfRange) {
		t.Fatalf("expected hour out of range error, got %v", err)
	}
	_, err = NewStoreTime(0, 60, 0)
	if !errors.Is(err, ErrMinuteOutOfRange) {
		t.Fatalf("expected minute out of range error, got %v", err)
	}
	_, err = NewStoreTime(0, 0, 60)
	if !errors.Is(err, ErrSecondOutOfRange) {
		t.Fatalf("expected second out of range error, got %v", err)
	}
	if _, err = NewStoreTime(23, 59, 59); err != nil {
		t.Fatalf("unexpected error for valid time: %v", err)
	}
}

func TestStoreTimeStringAndISO(t *testing.T) {
	st := MustStoreTime(5, 7, 9)
	if st.String() != "05:07:09" {
		t.Fatalf("unexpected String: %s", st.String())
	}
	if st.ISO() != "05:07:09" {
		t.Fatalf("unexpected ISO: %s", st.ISO())
	}
}

func TestStoreTimeComparison(t *testing.T) {
	a := MustStoreTime(9, 30, 0)
	b := MustStoreTime(9, 30, 1)
	c := MustStoreTime(10, 0, 0)

	if !a.Before(b) || b.Before(a) {
		t.Fatal("ordering between a and b incorrect")
	}
	if !b.Before(c) || !c.After(b) {
		t.Fatal("ordering between b and c incorrect")
	}
	if a.Equal(b) {
		t.Fatal("a and b should not be equal")
	}
	if !a.Equal(a) {
		t.Fatal("a should equal itself")
	}
}

func TestSecondsSinceMidnight(t *testing.T) {
	st := MustStoreTime(1, 1, 1)
	sec := st.SecondsSinceMidnight()
	if sec != 3600+60+1 {
		t.Fatalf("expected %d got %d", 3600+60+1, sec)
	}
}

func TestAddSecondsWrap(t *testing.T) {
	st := MustStoreTime(23, 59, 50)
	added, err := st.AddSeconds(15, true) // wrap
	if err != nil {
		t.Fatalf("wrap add failed: %v", err)
	}
	if added.String() != "00:00:05" {
		t.Fatalf("expected 00:00:05 got %s", added.String())
	}

	// No wrap should error
	if _, err := st.AddSeconds(15, false); err == nil {
		t.Fatal("expected error without wrap")
	}

	// Negative wrap
	st2 := MustStoreTime(0, 0, 5)
	back, err := st2.AddSeconds(-10, true)
	if err != nil {
		t.Fatalf("negative wrap failed: %v", err)
	}
	if back.String() != "23:59:55" {
		t.Fatalf("expected 23:59:55 got %s", back.String())
	}

	// Negative no wrap should error
	if _, err := st2.AddSeconds(-10, false); err == nil {
		t.Fatal("expected error for negative no-wrap")
	}
}

func TestSub(t *testing.T) {
	a := MustStoreTime(10, 0, 0)
	b := MustStoreTime(9, 30, 0)
	if diff := a.Sub(b); diff != 1800 {
		t.Fatalf("expected 1800 got %d", diff)
	}
	if diff := b.Sub(a); diff != -1800 {
		t.Fatalf("expected -1800 got %d", diff)
	}
}

func TestParseClock(t *testing.T) {
	st, err := ParseClock("13:05:07")
	if err != nil {
		t.Fatalf("parse clock failed: %v", err)
	}
	if st.String() != "13:05:07" {
		t.Fatalf("unexpected time: %s", st)
	}

	st2, err := ParseClock("08:15")
	if err != nil {
		t.Fatalf("parse clock HH:MM failed: %v", err)
	}
	if st2.String() != "08:15:00" {
		t.Fatalf("expected 08:15:00 got %s", st2)
	}

	if _, err := ParseClock("8"); err == nil {
		t.Fatal("expected error for invalid format")
	}
}

func TestParseFlexibleTime(t *testing.T) {
	tests := map[string]string{
		"13:05:07": "13:05:07",
		"13:05":    "13:05:00",
		"130507":   "13:05:07",
		"1305":     "13:05:00",
	}
	for in, want := range tests {
		st, err := ParseFlexibleTime(in)
		if err != nil {
			t.Fatalf("ParseFlexibleTime(%q) error: %v", in, err)
		}
		if got := st.String(); got != want {
			t.Fatalf("ParseFlexibleTime(%q) expected %s got %s", in, want, got)
		}
	}

	if _, err := ParseFlexibleTime(""); err == nil {
		t.Fatal("expected error for empty input")
	}
	if _, err := ParseFlexibleTime("invalid"); err == nil {
		t.Fatal("expected error for invalid digits")
	}
}

func TestStoreTimeFromTime(t *testing.T) {
	now := time.Date(2025, 12, 24, 9, 8, 7, 0, time.UTC)
	st := StoreTimeFromTime(now)
	if st.String() != "09:08:07" {
		t.Fatalf("unexpected extracted time: %s", st)
	}
}

func TestToTime(t *testing.T) {
	date := MustStoreDate(2025, 12, 24)
	st := MustStoreTime(9, 30, 15)
	ts := st.ToTime(date)
	if ts.Year() != 2025 || ts.Month() != 12 || ts.Day() != 24 || ts.Hour() != 9 || ts.Minute() != 30 || ts.Second() != 15 {
		t.Fatalf("unexpected time composition: %v", ts)
	}

	loc, _ := time.LoadLocation("UTC")
	ts2 := st.ToTimeIn(date, loc)
	if !ts.Equal(ts2) {
		t.Fatalf("expected same instant got %v vs %v", ts, ts2)
	}
}

func TestJSONSpacingExact(t *testing.T) {
	st := MustStoreTime(0, 5, 9)
	b, err := json.Marshal(st)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != "[0,5,9]" {
		t.Fatalf("expected compact array got %s", b)
	}
}
