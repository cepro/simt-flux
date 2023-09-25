package timeutils

import (
	"sort"
	"time"
)

type ClockTimePeriod struct {
	Start ClockTime `json:"start"`
	End   ClockTime `json:"end"`
}

// Contains returns True if t is within the ClockTimePeriod
func (p *ClockTimePeriod) Contains(t time.Time) bool {

	year, month, day := t.Date()

	// TODO: support periods that span over midnight

	startDateTime := p.Start.OnDate(year, month, day)
	endDateTime := p.End.OnDate(year, month, day)

	return (startDateTime.Before(t) && endDateTime.After(t)) || t.Equal(startDateTime) || t.Equal(endDateTime)
}

// NextStartTimes returns the next times at which the given ClockTimePeriods will start, given the current time as `t`.
// Absolute time.Time instances are calculated from the 'relative clock times', all of which are guaranteed to be in the future (relative to `t`).
// The returned times are sorted into ascending order.
func NextStartTimes(t time.Time, periods []ClockTimePeriod) []time.Time {
	startTimes := make([]time.Time, 0, len(periods))
	for _, period := range periods {

		// convert the 'relative clock time' into an 'absolute time'
		start := time.Date(t.Year(), t.Month(), t.Day(), period.Start.Hour, period.Start.Minute, period.Start.Second, 0, period.Start.Location)

		// if the 'absolute time' is in the past, push it into the future by adding a day
		if start.Before(t) {
			start = start.AddDate(0, 0, 1)
		}

		startTimes = append(startTimes, start)
	}

	// sort the startTimes into ascending order
	sort.Slice(startTimes, func(i, j int) bool {
		return startTimes[i].Before(startTimes[j])
	})

	return startTimes
}
