package timeutils

import (
	"time"
)

// DayedPeriod gives a period of time on particular days
type DayedPeriod struct {
	ClockTimePeriod `yaml:",inline"` // The period in clock time, e.g. "4pm to 6pm"
	Days            Days             `yaml:"days"` // Indicates the days on which this period applies
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

	if !d.Days.IsOnDay(t) {
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
