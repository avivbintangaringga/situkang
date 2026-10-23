// Package clock provides the application clock so tests can fix "now".
package clock

import "time"

// Clock reports the current time in the application timezone.
type Clock interface {
	Now() time.Time
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
