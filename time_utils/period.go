package timeutils

import "time"

// Period represents an absolute period between two instances in time, e.g. "2023/10/19 16:00:00 to 2023/10/19 18:00:00".
type Period struct {
	Start time.Time
	End   time.Time
}

// Equal returns true if the two period instances contain the same start and end times.
// These may be in different timezones but must be at the same instant in time.
// See the documentation on the Time type for the pitfalls of using == with Time values; most code should use Equal instead.
func (p Period) Equal(p2 Period) bool {
	return p.Start.Equal(p2.Start) && p.End.Equal(p2.End)
}

// Contains returns true if `t` is within the Period, inclusive of `Start` but exclusive of `End`.
func (p Period) Contains(t time.Time) bool {
	return (p.Start.Before(t) && p.End.After(t)) || p.Start.Equal(t)
}
