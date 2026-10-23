// Package clock provides the application clock so tests can fix "now".
package clock

import (
	"fmt"
	"time"
)

// DateLayout is the only accepted date format (date-only, no time).
const DateLayout = "2006-01-02"

// Clock reports the current time in the application timezone.
type Clock interface {
	Now() time.Time
	Today() time.Time
}

type realClock struct {
	loc *time.Location
}

// New returns a clock bound to loc. A nil location falls back to UTC.
func New(loc *time.Location) Clock {
	if loc == nil {
		loc = time.UTC
	}
	return realClock{loc: loc}
}

func (c realClock) Now() time.Time {
	return time.Now().In(c.loc)
}

func (c realClock) Today() time.Time {
	return midnight(c.Now())
}

type fixedClock struct {
	t time.Time
}

// Fixed returns a clock stuck at t. Use it in tests.
func Fixed(t time.Time) Clock {
	return fixedClock{t: t}
}

func (f fixedClock) Now() time.Time {
	return f.t
}

func (f fixedClock) Today() time.Time {
	return midnight(f.t)
}

func midnight(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// ParseDate parses a YYYY-MM-DD string and rejects impossible dates.
func ParseDate(value string) (time.Time, error) {
	t, err := time.Parse(DateLayout, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q: %w", value, err)
	}
	return t, nil
}

// FormatDate renders a date as YYYY-MM-DD.
func FormatDate(t time.Time) string {
	return t.Format(DateLayout)
}

// LastMonday returns the Monday of the week containing today (Monday-based).
func LastMonday(c Clock) time.Time {
	today := c.Today()
	offset := (int(today.Weekday()) + 6) % 7 // Monday = 0 ... Sunday = 6
	return today.AddDate(0, 0, -offset)
}

// WeekStart returns the Monday of the week containing t.
func WeekStart(t time.Time) time.Time {
	offset := (int(t.Weekday()) + 6) % 7
	return midnight(t).AddDate(0, 0, -offset)
}
