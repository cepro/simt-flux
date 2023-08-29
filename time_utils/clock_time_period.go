package timeutils

import "time"

type ClockTimePeriod struct {
	Start ClockTime
	End   ClockTime
}

// Contains returns True if t is within the ClockTimePeriod
func (p *ClockTimePeriod) Contains(t time.Time) bool {

	year, month, day := t.Date()

	// TODO: support periods that span over midnight

	startDateTime := p.Start.OnDate(year, month, day)
	endDateTime := p.End.OnDate(year, month, day)

	return (startDateTime.Before(t) && endDateTime.After(t)) || t.Equal(startDateTime) || t.Equal(endDateTime)

}
