package controller

import (
	"time"

	timeutils "github.com/cepro/besscontroller/time_utils"
)

type TimedCharge struct {
	Rate           float64 // p/kWh
	PeriodsWeekday []timeutils.ClockTimePeriod
	PeriodsWeekend []timeutils.ClockTimePeriod
}

// perKwhRate returns the applicable p/kWh rate and a boolean indicating if the rate applies to the given time or not.
func (d *TimedCharge) perKwhRate(t time.Time) (float64, bool) {
	periods := d.PeriodsWeekend
	if timeutils.IsWeekday(t) {
		periods = d.PeriodsWeekday
	}

	for _, period := range periods {
		if period.Contains(t) {
			return d.Rate, true
		}
	}
	return 0, false
}

// firstTimedCharges returns the first of the given charges that apply for the given `t` if one was found, and a boolean indicating if an applicable charge was found.
func firstTimedCharges(t time.Time, charges []TimedCharge) (float64, bool) {
	for _, charge := range charges {
		rate, found := charge.perKwhRate(t)
		if found {
			return rate, true
		}
	}
	return 0, false
}

// sumTimedCharges returns the sum of the given charges that apply for the given `t`.
func sumTimedCharges(t time.Time, charges []TimedCharge) float64 {
	total := 0.0

	for _, charge := range charges {
		rate, found := charge.perKwhRate(t)
		if found {
			total += rate
		}
	}
	return total
}
