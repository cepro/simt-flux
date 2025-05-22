package timeutils

import "time"

// IsWeekday returns true if the day is Mon-Fri inclusive, or False if the day is Sat or Sun
func IsWeekday(t time.Time) bool {
	day := t.Weekday()
	if day == time.Saturday || day == time.Sunday {
		return false
	}
	return true
}
