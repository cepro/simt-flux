package timeutils

import (
	"time"
)

// ClockTimePeriod represents a period of time that is defined by local clock time, without any date information,  e.g. "4pm to 6pm".
type ClockTimePeriod struct {
	Start ClockTime `yaml:"start"`
	End   ClockTime `yaml:"end"`
}

// AbsolutePeriod returns the equivilent `Period` instance for the given `ClockTimePeriod`, using `t` as the
// reference time that must be within the `ClockTimePeriod`.
// If `t` is outside of the `ClockTimePeriod` then the `ok` boolean is returned as false.
//
// This function is inclusive of the Period.Start, but exclusive of the Period.End.
//
// For example, calling on a ClockTimePeriod of "4pm to 6pm" using a reference `t` of "2023/10/19 16:53:00" would
// yield the period: "2023/10/19 16:00:00 to 2023/10/19 18:00:00".
//
// Another example, calling on a ClockTimePeriod of "4pm to 6pm" using a reference `t` of "2023/10/19 10:00:00" would
// result in false being returned as the given time is outside of the ClockTimePeriod.
func (p *ClockTimePeriod) AbsolutePeriod(t time.Time) (Period, bool) {

	if p.Start.Location.String() != p.End.Location.String() {
		// TODO: using the String() method here is not great- perhaps there is a better way of comparing time.Location instances?
		panic("Clock time period must start and end in the same timezone")
	}

	msStart := p.Start.Hour*int(time.Hour) + p.Start.Minute*int(time.Minute) + p.Start.Second*int(time.Second)
	msEnd := p.End.Hour*int(time.Hour) + p.End.Minute*int(time.Minute) + p.End.Second*int(time.Second)
	if msEnd < msStart {
		panic("Clock time period must end after it starts")
		// We do not currently support periods that cross midnight
	}

	// Make sure that `t` is in the relevant timezone for the ClockTimePeriod configuration, otherwise the day can be wrong
	// if it is near midnight and there is a timezone offset
	t = t.In(p.Start.Location)
	year, month, day := t.Date()

	startDateTime := p.Start.OnDate(year, month, day)
	endDateTime := p.End.OnDate(year, month, day)

	isContained := (startDateTime.Before(t) && endDateTime.After(t)) || t.Equal(startDateTime)

	if !isContained {
		return Period{}, false
	}

	return Period{Start: startDateTime, End: endDateTime}, true
}

// AbsolutePeriodOnDate returns the equivilent `Period` instance for the given `ClockTimePeriod` that occurs on the given date
func (p *ClockTimePeriod) AbsolutePeriodOnDate(year int, month time.Month, day int) Period {
	start := time.Date(year, month, day, p.Start.Hour, p.Start.Minute, p.Start.Second, 0, p.Start.Location)
	end := time.Date(year, month, day, p.End.Hour, p.End.Minute, p.End.Second, 0, p.End.Location)
	return Period{
		Start: start,
		End:   end,
	}
}

// Contains returns true if the given t is contained in the ClockTimePeriod
func (p *ClockTimePeriod) Contains(t time.Time) bool {
	_, contains := p.AbsolutePeriod(t)
	return contains
}
