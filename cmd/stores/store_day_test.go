package main

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestStoreDayRoundTrip(t *testing.T) {
	open := MustStoreTime(9, 0, 0)
	close := MustStoreTime(17, 30, 0)
	day := MustStoreDay(open, close)

	b, err := json.Marshal(day)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	expJSON := "[[9,0,0],[17,30,0]]"
	if string(b) != expJSON {
		t.Fatalf("expected %s got %s", expJSON, string(b))
	}

	var got StoreDay
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !got.Open.Equal(open) || !got.Close.Equal(close) {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}

func TestStoreDayValidation(t *testing.T) {
	// Invalid order (open >= close)
	_, err := NewStoreDay(MustStoreTime(10, 0, 0), MustStoreTime(10, 0, 0))
	if !errors.Is(err, ErrInvalidStoreDayOrder) {
		t.Fatalf("expected ErrInvalidStoreDayOrder got %v", err)
	}

	_, err = NewStoreDay(MustStoreTime(11, 0, 0), MustStoreTime(10, 0, 0))
	if !errors.Is(err, ErrInvalidStoreDayOrder) {
		t.Fatalf("expected ErrInvalidStoreDayOrder (open after close) got %v", err)
	}

	// Invalid open time
	badOpen := StoreTime{Hour: 25, Minute: 0, Second: 0}
	_, err = NewStoreDay(badOpen, MustStoreTime(12, 0, 0))
	if err == nil {
		t.Fatal("expected error for invalid open time")
	}

	// Valid
	if _, err := NewStoreDay(MustStoreTime(0, 0, 0), MustStoreTime(0, 0, 1)); err != nil {
		t.Fatalf("unexpected error valid day: %v", err)
	}
}

func TestStoreDayDurationAndContains(t *testing.T) {
	d := MustStoreDay(MustStoreTime(9, 15, 0), MustStoreTime(10, 0, 0))
	if secs := d.DurationSeconds(); secs != (45 * 60) {
		t.Fatalf("expected 2700 seconds got %d", secs)
	}

	tests := []struct {
		h, m, s int
		want    bool
	}{
		{9, 15, 0, true},   // boundary inclusive
		{9, 14, 59, false}, // before
		{9, 59, 59, true},  // inside
		{10, 0, 0, false},  // close boundary exclusive
	}

	for _, tc := range tests {
		tm := MustStoreTime(tc.h, tc.m, tc.s)
		if got := d.Contains(tm); got != tc.want {
			t.Fatalf("Contains(%s) = %v want %v", tm, got, tc.want)
		}
	}
}

func TestStoreDayOverlapsAdjacentMerge(t *testing.T) {
	a := MustStoreDay(MustStoreTime(9, 0, 0), MustStoreTime(11, 0, 0))
	b := MustStoreDay(MustStoreTime(10, 30, 0), MustStoreTime(12, 0, 0)) // overlap
	c := MustStoreDay(MustStoreTime(12, 0, 0), MustStoreTime(13, 0, 0))  // adjacent to merged(a,b)
	d := MustStoreDay(MustStoreTime(14, 0, 0), MustStoreTime(15, 0, 0))  // disjoint

	if !a.Overlaps(b) {
		t.Fatal("expected overlap a & b")
	}
	if a.Overlaps(c) {
		t.Fatal("did not expect overlap a & c")
	}
	if b.Overlaps(c) {
		t.Fatal("did not expect overlap b & c (they are adjacent)")
	}
	if !b.Adjacent(c) {
		t.Fatal("expected adjacency between b & c")
	}

	if merged, ok := a.Merge(b); !ok || merged.Open.String() != "09:00:00" || merged.Close.String() != "12:00:00" {
		t.Fatalf("merge a,b failed: %v %v", merged, ok)
	}

	mergedAB, _ := a.Merge(b)
	if !mergedAB.Adjacent(c) {
		t.Fatal("expected mergedAB adjacent to c")
	}

	if mergedAll, ok := mergedAB.Merge(c); !ok || mergedAll.Close.String() != "13:00:00" {
		t.Fatalf("merge mergedAB,c failed: %v %v", mergedAll, ok)
	}

	if _, ok := a.Merge(d); ok {
		t.Fatal("expected no merge with disjoint interval")
	}
}

func TestStoreDaySplit(t *testing.T) {
	day := MustStoreDay(MustStoreTime(8, 0, 0), MustStoreTime(12, 0, 0))
	splitTime := MustStoreTime(10, 0, 0)

	left, right, ok := day.Split(splitTime)
	if !ok {
		t.Fatal("expected successful split")
	}
	if left.Open.String() != "08:00:00" || left.Close.String() != "10:00:00" {
		t.Fatalf("unexpected left: %+v", left)
	}
	if right.Open.String() != "10:00:00" || right.Close.String() != "12:00:00" {
		t.Fatalf("unexpected right: %+v", right)
	}

	// Split at open -> no split
	if _, _, ok := day.Split(day.Open); ok {
		t.Fatal("expected no split at open boundary")
	}

	// Outside interval
	if _, _, ok := day.Split(MustStoreTime(7, 59, 59)); ok {
		t.Fatal("expected no split for time before interval")
	}
	if _, _, ok := day.Split(MustStoreTime(12, 0, 0)); ok {
		t.Fatal("expected no split for time at close boundary")
	}
}

func TestStoreDayClamp(t *testing.T) {
	base := MustStoreDay(MustStoreTime(9, 0, 0), MustStoreTime(17, 0, 0))
	bounds := MustStoreDay(MustStoreTime(8, 30, 0), MustStoreTime(12, 0, 0))

	clamped, ok := base.Clamp(bounds)
	if !ok {
		t.Fatal("expected clamp success")
	}
	if clamped.Open.String() != "09:00:00" || clamped.Close.String() != "12:00:00" {
		t.Fatalf("unexpected clamped: %+v", clamped)
	}

	disjointBounds := MustStoreDay(MustStoreTime(17, 0, 0), MustStoreTime(18, 0, 0))
	if _, ok := base.Clamp(disjointBounds); ok {
		t.Fatal("expected clamp failure for disjoint bounds")
	}
}

func TestNormalizeStoreDays(t *testing.T) {
	// Unsorted with overlaps & adjacency:
	// [09:00-10:00], [09:30-11:00] -> merge to [09:00-11:00]
	// [11:00-12:00] adjacent to merged -> merge to [09:00-12:00]
	// [13:00-14:00] separate
	input := []StoreDay{
		MustStoreDay(MustStoreTime(11, 0, 0), MustStoreTime(12, 0, 0)),
		MustStoreDay(MustStoreTime(9, 30, 0), MustStoreTime(11, 0, 0)),
		MustStoreDay(MustStoreTime(9, 0, 0), MustStoreTime(10, 0, 0)),
		MustStoreDay(MustStoreTime(13, 0, 0), MustStoreTime(14, 0, 0)),
	}

	out := NormalizeStoreDays(input)
	if len(out) != 2 {
		t.Fatalf("expected 2 normalized intervals got %d: %+v", len(out), out)
	}
	if out[0].Open.String() != "09:00:00" || out[0].Close.String() != "12:00:00" {
		t.Fatalf("unexpected first merged interval: %+v", out[0])
	}
	if out[1].Open.String() != "13:00:00" || out[1].Close.String() != "14:00:00" {
		t.Fatalf("unexpected second interval: %+v", out[1])
	}
}

func TestStoreDayJSONErrors(t *testing.T) {
	// Outer length wrong
	var d StoreDay
	if err := json.Unmarshal([]byte("[[9,0,0]]"), &d); err == nil {
		t.Fatal("expected error for wrong outer length")
	}

	// Inner length wrong
	if err := json.Unmarshal([]byte("[[9,0],[10,0,0]]"), &d); err == nil {
		t.Fatal("expected error for wrong inner length (open)")
	}
	if err := json.Unmarshal([]byte("[[9,0,0],[10,0]]"), &d); err == nil {
		t.Fatal("expected error for wrong inner length (close)")
	}

	// Invalid order
	if err := json.Unmarshal([]byte("[[10,0,0],[9,0,0]]"), &d); err == nil {
		t.Fatal("expected error for inverted times")
	}
}
