package config

import (
	"time"

	timeutils "github.com/cepro/besscontroller/time_utils"
)

// TimedRate represents a p/kWh that only applies at certain times of day
type TimedRate struct {
	Rate    float64                 `yaml:"rate"`
	Periods []timeutils.DayedPeriod `yaml:"periods"`
}

// perKwhRate returns the applicable p/kWh rate and a boolean indicating if the rate applies to the given time or not.
func (r *TimedRate) perKwhRate(t time.Time) (float64, bool) {

	for _, dayedPeriod := range r.Periods {
		if dayedPeriod.Contains(t) {
			return r.Rate, true
		}
	}
	return 0, false
}

// FirstTimedRate returns the first of the given charges that apply for the given `t` if one was found, and a boolean indicating if an applicable charge was found.
func FirstTimedRate(t time.Time, charges []TimedRate) (float64, bool) {
	for _, charge := range charges {
		rate, found := charge.perKwhRate(t)
		if found {
			return rate, true
		}
	}
	return 0, false
}

// SumTimedRates returns the sum of the given charges that apply for the given `t`.
func SumTimedRates(t time.Time, charges []TimedRate) float64 {
	total := 0.0

	for _, charge := range charges {
		rate, found := charge.perKwhRate(t)
		if found {
			total += rate
		}
	}
	return total
}
