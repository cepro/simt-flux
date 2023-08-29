package timeutils

import "time"

// ClockTime represents a time of day in the given locale, without a date.
type ClockTime struct {
	Hour     int
	Minute   int
	Second   int
	Location *time.Location
}

// OnDate returns a time with the given clock time on the given date
func (c *ClockTime) OnDate(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, c.Hour, c.Minute, c.Second, 0, c.Location)
}
