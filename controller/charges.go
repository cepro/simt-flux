package controller

import (
	"time"

	timeutils "github.com/cepro/besscontroller/time_utils"
)

type DuosCharge struct {
	Rate           float64 // p/kWh
	PeriodsWeekday []timeutils.ClockTimePeriod
	PeriodsWeekend []timeutils.ClockTimePeriod
}

func (d *DuosCharge) perKwhRate(t time.Time) float64 {
	periods := d.PeriodsWeekend
	if timeutils.IsWeekday(t) {
		periods = d.PeriodsWeekday
	}

	for _, period := range periods {
		if period.Contains(t) {
			return d.Rate
		}
	}
	return 0
}

// getDuosCharges returns the total charge for the given `t`
func getDuosCharges(t time.Time, charges []DuosCharge) float64 {
	total := 0.0
	for _, charge := range charges {
		total += charge.perKwhRate(t)
	}
	return total
}
