package timeutils

import (
	"fmt"
	"time"
)

// Days is a string representation of the different ways to configure days. At the moment, only a few options are
// required, but we could allow any combination of days in the future. E.g. with a string like "weekends|monday|wednesday".
type Days string

const (
	WeekendDays = "weekends"
	WeekdayDays = "weekdays"
	AllDays     = "all"
)

// DayedPeriod gives a period of time on particular days
type DayedPeriod struct {
	ClockTimePeriod      // The period in clock time, e.g. "4pm to 6pm"
	Days            Days `json:"days"` // Indicates the days on which this period applies, can be "weekends", "weekdays", or "all"
}

// AbsolutePeriod returns the equivilent `Period` instance for the given `DayedPeriod`, using `t` as the
// reference time that must be within the `DayedPeriod`.
// If `t` is outside of the `DayedPeriod` (i.e. on the wrong day or at the wrong time) then the `ok` boolean is returned as false.
//
// This function is inclusive of the Period.Start, but exclusive of the Period.End.
//
// For example, calling on a DayedPeriod of "4pm to 6pm on all days" using a reference `t` of "2023/10/19 16:53:00" would
// yield the period: "2023/10/19 16:00:00 to 2023/10/19 18:00:00".
//
// Another example, calling on a DayedPeriod of "4pm to 6pm on Saturdays" using a reference `t` of "2023/10/19 5pm (thursday)" would
// result in false being returned as the given time is on the wrong day (even though it's at the right time of day).
//
// Another example, calling on a DayedPeriod of "4pm to 6pm on all days" using a reference `t` of "2023/10/19 10:00:00" would
// result in false being returned as the given time is at the wrong time of day (even though the day itself is okay).
func (d *DayedPeriod) AbsolutePeriod(t time.Time) (Period, bool) {

	if !d.IsOnDay(t) {
		return Period{}, false
	}

	// Now that we know the day is okay, we can use the ClockTimePeriod's AbsolutePeriod function
	return d.ClockTimePeriod.AbsolutePeriod(t)
}

// Contains returns true if the given t is contained in the DayedPeriod
func (d *DayedPeriod) Contains(t time.Time) bool {
	_, contains := d.AbsolutePeriod(t)
	return contains
}

func (d *DayedPeriod) IsOnDay(t time.Time) bool {
	switch d.Days {
	case AllDays:
		return true // the day is always okay
	case WeekdayDays:
		if IsWeekday(t) {
			return true
		} else {
			return false
		}
	case WeekendDays:
		if !IsWeekday(t) {
			return true
		} else {
			return false
		}
	default:
		panic(fmt.Sprintf("Unknown day specification: '%s'", d.Days))
	}
}
