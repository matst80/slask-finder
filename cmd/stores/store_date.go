package main

/*
 * Store date, always 3 integers in an array
 * [2025, 12, 24]
 */

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// StoreDate represents a calendar date (no time-of-day, no timezone).
// It is serialized to JSON strictly as: [year, month, day].
type StoreDate struct {
	Year  int
	Month int
	Day   int
}

// Error variables you can sentinel-match against (wrapped by Validate()).
var (
	ErrYearOutOfRange  = errors.New("storedate: year out of range (1..9999)")
	ErrMonthOutOfRange = errors.New("storedate: month out of range (1..12)")
	ErrDayOutOfRange   = errors.New("storedate: day out of range for month")
	ErrBadArrayLength  = errors.New("storedate: expected JSON array length 3")
)

// NewStoreDate validates and returns a StoreDate.
func NewStoreDate(year, month, day int) (StoreDate, error) {
	sd := StoreDate{Year: year, Month: month, Day: day}
	if err := sd.Validate(); err != nil {
		return StoreDate{}, err
	}
	return sd, nil
}

// MustStoreDate panics on invalid date (convenience for constants / tests).
func MustStoreDate(year, month, day int) StoreDate {
	sd, err := NewStoreDate(year, month, day)
	if err != nil {
		panic(err)
	}
	return sd
}

// FromTime constructs a StoreDate from a time.Time (in that time's calendar fields).
func FromTime(t time.Time) StoreDate {
	return StoreDate{
		Year:  t.Year(),
		Month: int(t.Month()),
		Day:   t.Day(),
	}
}

// Today returns today's date in the provided location (or time.Local if nil).
func Today(loc *time.Location) StoreDate {
	if loc == nil {
		loc = time.Local
	}
	now := time.Now().In(loc)
	return FromTime(now)
}

// IsZero returns true if all fields are zero (uninitialized struct).
func (sd StoreDate) IsZero() bool {
	return sd.Year == 0 && sd.Month == 0 && sd.Day == 0
}

// Validate checks the date fields for logical correctness.
func (sd StoreDate) Validate() error {
	if sd.Year < 1 || sd.Year > 9999 {
		return fmt.Errorf("%w: %d", ErrYearOutOfRange, sd.Year)
	}
	if sd.Month < 1 || sd.Month > 12 {
		return fmt.Errorf("%w: %d", ErrMonthOutOfRange, sd.Month)
	}
	dim := daysInMonth(sd.Year, sd.Month)
	if sd.Day < 1 || sd.Day > dim {
		return fmt.Errorf("%w: got %d, max %d (year=%d month=%d)", ErrDayOutOfRange, sd.Day, dim, sd.Year, sd.Month)
	}
	return nil
}

// Time returns a time.Time at midnight UTC for this date.
// (You can use In(loc) or .Local() to shift as needed.)
func (sd StoreDate) Time() time.Time {
	return time.Date(sd.Year, time.Month(sd.Month), sd.Day, 0, 0, 0, 0, time.UTC)
}

// TimeIn returns a time.Time at midnight in the specified location (or UTC if nil).
func (sd StoreDate) TimeIn(loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	return time.Date(sd.Year, time.Month(sd.Month), sd.Day, 0, 0, 0, 0, loc)
}

// Date components (year, month, day) similar to time.Time.Date().
func (sd StoreDate) Date() (int, time.Month, int) {
	return sd.Year, time.Month(sd.Month), sd.Day
}

// ISO returns the canonical YYYY-MM-DD string.
func (sd StoreDate) ISO() string {
	return fmt.Sprintf("%04d-%02d-%02d", sd.Year, sd.Month, sd.Day)
}

// String implements fmt.Stringer (same as ISO()).
func (sd StoreDate) String() string {
	return sd.ISO()
}

// AddDays returns a new StoreDate offset by n days (can be negative).
func (sd StoreDate) AddDays(n int) StoreDate {
	t := sd.Time().AddDate(0, 0, n)
	return FromTime(t)
}

// DaysSince returns the number of days (signed) from other -> sd.
func (sd StoreDate) DaysSince(other StoreDate) int {
	// Convert both to UTC midnight and subtract.
	dur := sd.Time().Sub(other.Time())
	return int(dur.Hours() / 24)
}

// Before returns true if sd is chronologically before other.
func (sd StoreDate) Before(other StoreDate) bool {
	if sd.Year != other.Year {
		return sd.Year < other.Year
	}
	if sd.Month != other.Month {
		return sd.Month < other.Month
	}
	return sd.Day < other.Day
}

// After returns true if sd is chronologically after other.
func (sd StoreDate) After(other StoreDate) bool {
	if sd.Year != other.Year {
		return sd.Year > other.Year
	}
	if sd.Month != other.Month {
		return sd.Month > other.Month
	}
	return sd.Day > other.Day
}

// Equal compares for exact date equality.
func (sd StoreDate) Equal(other StoreDate) bool {
	return sd.Year == other.Year && sd.Month == other.Month && sd.Day == other.Day
}

// MarshalJSON implements json.Marshaler to encode as [year,month,day].
func (sd StoreDate) MarshalJSON() ([]byte, error) {
	if err := sd.Validate(); err != nil {
		return nil, err
	}
	// Manual formatting avoids allocations from json.Marshal([]int{...})
	buf := []byte(fmt.Sprintf("[%d,%d,%d]", sd.Year, sd.Month, sd.Day))
	return buf, nil
}

// UnmarshalJSON implements json.Unmarshaler to decode from [year,month,day].
func (sd *StoreDate) UnmarshalJSON(data []byte) error {
	// Fast path: ensure it starts with '['
	trim := strings.TrimSpace(string(data))
	if len(trim) == 0 || trim[0] != '[' {
		return fmt.Errorf("storedate: expected JSON array, got: %s", string(data))
	}

	// Decode into a temporary slice
	var arr []int
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("storedate: invalid array form: %w", err)
	}
	if len(arr) != 3 {
		return fmt.Errorf("%w (got length %d)", ErrBadArrayLength, len(arr))
	}
	tmp := StoreDate{Year: arr[0], Month: arr[1], Day: arr[2]}
	if err := tmp.Validate(); err != nil {
		return err
	}
	*sd = tmp
	return nil
}

// ParseISO parses a YYYY-MM-DD string into a StoreDate.
func ParseISO(s string) (StoreDate, error) {
	parts := strings.Split(s, "-")
	if len(parts) != 3 {
		return StoreDate{}, fmt.Errorf("storedate: invalid ISO date %q", s)
	}
	y, err := strconv.Atoi(parts[0])
	if err != nil {
		return StoreDate{}, fmt.Errorf("storedate: bad year: %w", err)
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil {
		return StoreDate{}, fmt.Errorf("storedate: bad month: %w", err)
	}
	d, err := strconv.Atoi(parts[2])
	if err != nil {
		return StoreDate{}, fmt.Errorf("storedate: bad day: %w", err)
	}
	return NewStoreDate(y, m, d)
}

// ParseFlexible tries ISO first, then RFC3339 date portion (YYYY-MM-DDTHH...).
func ParseFlexible(s string) (StoreDate, error) {
	if len(s) >= 10 && (strings.Contains(s, "T") || strings.Contains(s, "t")) {
		// Try time.Parse for broader inputs
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return FromTime(t), nil
		}
		// Fallback: take first 10 chars
		return ParseISO(s[:10])
	}
	return ParseISO(s)
}

// daysInMonth returns number of days in a given month/year.
func daysInMonth(year, month int) int {
	// Use time.Date trick: day 0 of next month = last day of this month
	t := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC)
	return t.Day()
}
