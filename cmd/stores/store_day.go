package main

/*
 * StoreDay represents an opening interval during a single day as exactly two StoreTime
 * values inside an array:
 *
 *   JSON: [[openHour,openMinute,openSecond],[closeHour,closeMinute,closeSecond]]
 *
 * Example:
 *   [[9,0,0],[18,30,0]]  -> open 09:00:00 close 18:30:00
 *
 * This mirrors the compact array representation used for StoreDate ([Y,M,D]) and
 * StoreTime ([H,M,S]).
 */

import (
	"encoding/json"
	"errors"
	"fmt"
)

// StoreDay holds an opening interval from Open (inclusive) to Close (exclusive).
type StoreDay struct {
	Open  StoreTime
	Close StoreTime
}

// Sentinel errors.
var (
	ErrBadStoreDayArrayLength = errors.New("storeday: expected JSON outer array length 2")
	ErrBadStoreDayInnerLength = errors.New("storeday: expected inner time array length 3")
	ErrInvalidStoreDayOrder   = errors.New("storeday: open time must be strictly before close time")
)

// NewStoreDay constructs and validates a StoreDay.
func NewStoreDay(open, close StoreTime) (StoreDay, error) {
	sd := StoreDay{Open: open, Close: close}
	if err := sd.Validate(); err != nil {
		return StoreDay{}, err
	}
	return sd, nil
}

// MustStoreDay panics on invalid data (useful in tests / constants).
func MustStoreDay(open, close StoreTime) StoreDay {
	d, err := NewStoreDay(open, close)
	if err != nil {
		panic(err)
	}
	return d
}

// Validate checks logical correctness:
// - Open & Close times individually valid
// - Open strictly before Close (disallow zero-length or inverted intervals)
func (sd StoreDay) Validate() error {
	if err := sd.Open.Validate(); err != nil {
		return fmt.Errorf("storeday: open invalid: %w", err)
	}
	if err := sd.Close.Validate(); err != nil {
		return fmt.Errorf("storeday: close invalid: %w", err)
	}
	if !sd.Open.Before(sd.Close) {
		return fmt.Errorf("%w: open=%s close=%s", ErrInvalidStoreDayOrder, sd.Open, sd.Close)
	}
	return nil
}

// DurationSeconds returns the number of seconds in the interval.
func (sd StoreDay) DurationSeconds() int {
	return sd.Close.Sub(sd.Open)
}

// Contains reports whether the provided time t is within the interval
// (inclusive of Open, exclusive of Close) i.e. Open <= t < Close.
func (sd StoreDay) Contains(t StoreTime) bool {
	return (sd.Open.Equal(t) || sd.Open.Before(t)) && t.Before(sd.Close)
}

// Overlaps reports whether this interval overlaps another at all.
func (sd StoreDay) Overlaps(other StoreDay) bool {
	// Two half-open intervals [a,b) and [c,d) overlap if a < d && c < b.
	return sd.Open.Before(other.Close) && other.Open.Before(sd.Close)
}

// Adjacent reports whether this interval directly abuts the other without
// overlapping (i.e. this.Close == other.Open or other.Close == this.Open).
func (sd StoreDay) Adjacent(other StoreDay) bool {
	return sd.Close.Equal(other.Open) || other.Close.Equal(sd.Open)
}

// Merge attempts to merge two intervals if they overlap or are adjacent.
// Returns merged interval and true on success, or zero value + false if disjoint.
func (sd StoreDay) Merge(other StoreDay) (StoreDay, bool) {
	if !(sd.Overlaps(other) || sd.Adjacent(other)) {
		return StoreDay{}, false
	}
	// Pick earliest open and latest close.
	open := sd.Open
	if other.Open.Before(open) {
		open = other.Open
	}
	close := sd.Close
	if other.Close.After(close) {
		close = other.Close
	}
	merged := StoreDay{Open: open, Close: close}
	_ = merged.Validate() // Should be valid by construction.
	return merged, true
}

// MarshalJSON implements json.Marshaler producing:
// [[openH,openM,openS],[closeH,closeM,closeS]]
func (sd StoreDay) MarshalJSON() ([]byte, error) {
	if err := sd.Validate(); err != nil {
		return nil, err
	}
	// Manually format to avoid intermediate slices/allocations.
	// Reuse StoreTime.String isn't desired because we need the array form.
	return []byte(
		fmt.Sprintf(
			"[[%d,%d,%d],[%d,%d,%d]]",
			sd.Open.Hour, sd.Open.Minute, sd.Open.Second,
			sd.Close.Hour, sd.Close.Minute, sd.Close.Second,
		),
	), nil
}

// UnmarshalJSON implements json.Unmarshaler expecting outer length 2
// and each inner length 3 with valid time components.
func (sd *StoreDay) UnmarshalJSON(data []byte) error {
	// Decode generically then validate shape.
	var raw [][]int
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("storeday: invalid array form: %w", err)
	}
	if len(raw) != 2 {
		return fmt.Errorf("%w (got %d)", ErrBadStoreDayArrayLength, len(raw))
	}
	openArr := raw[0]
	closeArr := raw[1]
	if len(openArr) != 3 || len(closeArr) != 3 {
		return fmt.Errorf("%w", ErrBadStoreDayInnerLength)
	}
	open := StoreTime{Hour: openArr[0], Minute: openArr[1], Second: openArr[2]}
	close := StoreTime{Hour: closeArr[0], Minute: closeArr[1], Second: closeArr[2]}
	tmp := StoreDay{Open: open, Close: close}
	if err := tmp.Validate(); err != nil {
		return err
	}
	*sd = tmp
	return nil
}

// Split breaks the interval into two at the provided time t if t is strictly
// inside the interval. If t is not strictly inside, returns original and false.
func (sd StoreDay) Split(t StoreTime) (StoreDay, StoreDay, bool) {
	if !sd.Contains(t) || t.Equal(sd.Open) {
		return StoreDay{}, StoreDay{}, false
	}
	// t < Close because Contains ensures t < Close.
	left := StoreDay{Open: sd.Open, Close: t}
	right := StoreDay{Open: t, Close: sd.Close}
	// Both should validate (t inside implies strict ordering).
	if err := left.Validate(); err != nil {
		return StoreDay{}, StoreDay{}, false
	}
	if err := right.Validate(); err != nil {
		return StoreDay{}, StoreDay{}, false
	}
	return left, right, true
}

// Clamp restricts the interval to the overlap with bounds. If no overlap,
// returns zero-value and false.
func (sd StoreDay) Clamp(bounds StoreDay) (StoreDay, bool) {
	if !sd.Overlaps(bounds) {
		return StoreDay{}, false
	}
	open := sd.Open
	if bounds.Open.After(open) {
		open = bounds.Open
	}
	close := sd.Close
	if bounds.Close.Before(close) {
		close = bounds.Close
	}
	clamped := StoreDay{Open: open, Close: close}
	if err := clamped.Validate(); err != nil {
		return StoreDay{}, false
	}
	return clamped, true
}

// NormalizeSlice merges overlapping / adjacent intervals in-place.
// It assumes all intervals are for the same logical day.
// Returns a new slice (does not mutate input slice elements).
func NormalizeStoreDays(days []StoreDay) []StoreDay {
	if len(days) == 0 {
		return nil
	}
	// Simple insertion sort by Open then merge pass (n expected small).
	sorted := make([]StoreDay, 0, len(days))
	for _, d := range days {
		// Validate each; skip invalid
		if d.Validate() != nil {
			continue
		}
		inserted := false
		for i, cur := range sorted {
			if d.Open.Before(cur.Open) {
				sorted = append(sorted, StoreDay{})
				copy(sorted[i+1:], sorted[i:])
				sorted[i] = d
				inserted = true
				break
			}
		}
		if !inserted {
			sorted = append(sorted, d)
		}
	}
	if len(sorted) == 0 {
		return nil
	}
	merged := []StoreDay{sorted[0]}
	for i := 1; i < len(sorted); i++ {
		last := merged[len(merged)-1]
		if m, ok := last.Merge(sorted[i]); ok {
			merged[len(merged)-1] = m
		} else {
			merged = append(merged, sorted[i])
		}
	}
	return merged
}
