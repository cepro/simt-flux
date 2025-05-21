package timeutils

import "time"

// FloorHH returns the given `t` rounded down to the nearest half-hour boundary
func FloorHH(t time.Time) time.Time {
	minute := t.Minute()
	if minute >= 30 {
		minute = 30
	} else {
		minute = 0
	}
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), minute, 0, 0, t.Location())
}
