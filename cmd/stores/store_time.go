package main

/*
 * Store time, always 3 integers in an array
 * [hour, minute, second]
 * Example: [13, 45, 0] => 13:45:00
 */

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// StoreTime represents a wall-clock time (no date, no timezone).
// JSON form: [hour, minute, second]
type StoreTime struct {
	Hour   int
	Minute int
	Second int
}

// Sentinel errors (wrapped by Validate()).
var (
	ErrHourOutOfRange       = errors.New("storetime: hour out of range (0..23)")
	ErrMinuteOutOfRange     = errors.New("storetime: minute out of range (0..59)")
	ErrSecondOutOfRange     = errors.New("storetime: second out of range (0..59)")
	ErrBadTimeArrayLength   = errors.New("storetime: expected JSON array length 3")
	ErrInvalidClockString   = errors.New("storetime: invalid clock string")
	ErrNegativeSecondAdjust = errors.New("storetime: negative result from adjustment")
)

// NewStoreTime constructs and validates a StoreTime.
func NewStoreTime(hour, minute, second int) (StoreTime, error) {
	st := StoreTime{Hour: hour, Minute: minute, Second: second}
	if err := st.Validate(); err != nil {
		return StoreTime{}, err
	}
	return st, nil
}

// MustStoreTime panics on invalid inputs (useful for constants / tests).
func MustStoreTime(hour, minute, second int) StoreTime {
	st, err := NewStoreTime(hour, minute, second)
	if err != nil {
		panic(err)
	}
	return st
}

// Validate checks the time components.
func (st StoreTime) Validate() error {
	if st.Hour < 0 || st.Hour > 23 {
		return fmt.Errorf("%w: %d", ErrHourOutOfRange, st.Hour)
	}
	if st.Minute < 0 || st.Minute > 59 {
		return fmt.Errorf("%w: %d", ErrMinuteOutOfRange, st.Minute)
	}
	if st.Second < 0 || st.Second > 59 {
		return fmt.Errorf("%w: %d", ErrSecondOutOfRange, st.Second)
	}
	return nil
}

// IsZero reports whether all components are zero (00:00:00).
func (st StoreTime) IsZero() bool {
	return st.Hour == 0 && st.Minute == 0 && st.Second == 0
}

// String implements fmt.Stringer (HH:MM:SS).
func (st StoreTime) String() string {
	return fmt.Sprintf("%02d:%02d:%02d", st.Hour, st.Minute, st.Second)
}

// ISO returns the same as String() for consistency with StoreDate.ISO.
func (st StoreTime) ISO() string { return st.String() }

// MarshalJSON encodes the time as [hour,minute,second].
func (st StoreTime) MarshalJSON() ([]byte, error) {
	if err := st.Validate(); err != nil {
		return nil, err
	}
	return []byte(fmt.Sprintf("[%d,%d,%d]", st.Hour, st.Minute, st.Second)), nil
}

// UnmarshalJSON decodes [hour,minute,second].
func (st *StoreTime) UnmarshalJSON(data []byte) error {
	trim := strings.TrimSpace(string(data))
	if len(trim) == 0 || trim[0] != '[' {
		return fmt.Errorf("storetime: expected JSON array, got: %s", string(data))
	}
	var arr []int
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("storetime: invalid array form: %w", err)
	}
	if len(arr) != 3 {
		return fmt.Errorf("%w (got length %d)", ErrBadTimeArrayLength, len(arr))
	}
	tmp := StoreTime{Hour: arr[0], Minute: arr[1], Second: arr[2]}
	if err := tmp.Validate(); err != nil {
		return err
	}
	*st = tmp
	return nil
}

// ParseClock parses "HH:MM:SS" or "HH:MM" into a StoreTime.
// Seconds default to 0 if omitted.
func ParseClock(s string) (StoreTime, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return StoreTime{}, fmt.Errorf("%w: empty", ErrInvalidClockString)
	}
	parts := strings.Split(s, ":")
	if len(parts) != 2 && len(parts) != 3 {
		return StoreTime{}, fmt.Errorf("%w: %q", ErrInvalidClockString, s)
	}
	toInt := func(p string) (int, error) {
		if p == "" {
			return 0, fmt.Errorf("%w: blank component", ErrInvalidClockString)
		}
		v, err := strconv.Atoi(p)
		if err != nil {
			return 0, fmt.Errorf("%w: %v", ErrInvalidClockString, err)
		}
		return v, nil
	}
	h, err := toInt(parts[0])
	if err != nil {
		return StoreTime{}, err
	}
	m, err := toInt(parts[1])
	if err != nil {
		return StoreTime{}, err
	}
	sec := 0
	if len(parts) == 3 {
		sec, err = toInt(parts[2])
		if err != nil {
			return StoreTime{}, err
		}
	}
	return NewStoreTime(h, m, sec)
}

// SecondsSinceMidnight returns total seconds from 00:00:00.
func (st StoreTime) SecondsSinceMidnight() int {
	return st.Hour*3600 + st.Minute*60 + st.Second
}

// AddSeconds returns a new StoreTime offset by n seconds (can be negative).
// WrapAround indicates whether to wrap at 24h (true) or return error (false) if out of range.
func (st StoreTime) AddSeconds(n int, wrapAround bool) (StoreTime, error) {
	base := st.SecondsSinceMidnight() + n
	if wrapAround {
		const day = 24 * 3600
		base = ((base % day) + day) % day // mod that handles negative
	} else {
		if base < 0 || base >= 24*3600 {
			return StoreTime{}, fmt.Errorf("%w: adjustment leads to out-of-day range", ErrNegativeSecondAdjust)
		}
	}
	h := base / 3600
	m := (base % 3600) / 60
	s := base % 60
	return StoreTime{Hour: h, Minute: m, Second: s}, nil
}

// Compare returns -1 if st < other, 0 if equal, +1 if st > other.
func (st StoreTime) Compare(other StoreTime) int {
	if st.Hour != other.Hour {
		if st.Hour < other.Hour {
			return -1
		}
		return 1
	}
	if st.Minute != other.Minute {
		if st.Minute < other.Minute {
			return -1
		}
		return 1
	}
	if st.Second != other.Second {
		if st.Second < other.Second {
			return -1
		}
		return 1
	}
	return 0
}

// Before reports st < other.
func (st StoreTime) Before(other StoreTime) bool { return st.Compare(other) < 0 }

// After reports st > other.
func (st StoreTime) After(other StoreTime) bool { return st.Compare(other) > 0 }

// Equal reports st == other.
func (st StoreTime) Equal(other StoreTime) bool { return st.Compare(other) == 0 }

// Sub returns the signed number of seconds (st - other).
func (st StoreTime) Sub(other StoreTime) int {
	return st.SecondsSinceMidnight() - other.SecondsSinceMidnight()
}

// ToTime combines a StoreDate with this StoreTime to produce a time.Time in UTC.
func (st StoreTime) ToTime(date StoreDate) time.Time {
	// If date is zero, return zero time (unix epoch) with components if valid.
	if date.IsZero() {
		return time.Date(0, 1, 1, st.Hour, st.Minute, st.Second, 0, time.UTC)
	}
	return time.Date(date.Year, time.Month(date.Month), date.Day, st.Hour, st.Minute, st.Second, 0, time.UTC)
}

// ToTimeIn combines a StoreDate with this StoreTime using provided location (UTC if nil).
func (st StoreTime) ToTimeIn(date StoreDate, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	if date.IsZero() {
		return time.Date(0, 1, 1, st.Hour, st.Minute, st.Second, 0, loc)
	}
	return time.Date(date.Year, time.Month(date.Month), date.Day, st.Hour, st.Minute, st.Second, 0, loc)
}

// FromTime extracts a StoreTime from a time.Time.
func StoreTimeFromTime(t time.Time) StoreTime {
	return StoreTime{Hour: t.Hour(), Minute: t.Minute(), Second: t.Second()}
}

// ParseFlexibleTime attempts more lenient parsing:
// - "HH:MM:SS"
// - "HH:MM"
// - "HHMMSS" (6 digits)
// - "HHMM" (4 digits)
// Returns error if none match.
// (Renamed from ParseFlexible to avoid collision with StoreDate's ParseFlexible)
func ParseFlexibleTime(s string) (StoreTime, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return StoreTime{}, fmt.Errorf("%w: empty", ErrInvalidClockString)
	}
	// Try colon forms
	if strings.Contains(s, ":") {
		return ParseClock(s)
	}
	// Raw digit forms
	if len(s) == 6 { // HHMMSS
		h, _ := strconv.Atoi(s[0:2])
		m, _ := strconv.Atoi(s[2:4])
		c, _ := strconv.Atoi(s[4:6])
		return NewStoreTime(h, m, c)
	}
	if len(s) == 4 { // HHMM
		h, _ := strconv.Atoi(s[0:2])
		m, _ := strconv.Atoi(s[2:4])
		return NewStoreTime(h, m, 0)
	}
	return StoreTime{}, fmt.Errorf("%w: %q", ErrInvalidClockString, s)
}
