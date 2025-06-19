package controller

import (
	"log/slog"
	"time"

	"github.com/cepro/besscontroller/config"
	timeutils "github.com/cepro/besscontroller/time_utils"
)

// findDayedPeriodContainingTime searches the list of dayed periods and returns the first one that is active at the time `t` (i.e.
// the first one whose period contains `t`), or nil if none was found. It also returns the associated absolute period.
func findDayedPeriodContainingTime(t time.Time, dayedPeriods []timeutils.DayedPeriod) (*timeutils.DayedPeriod, *timeutils.Period) {

	for _, dayedPeriod := range dayedPeriods {
		period, ok := dayedPeriod.AbsolutePeriod(t)
		if ok {
			return &dayedPeriod, &period
		}
	}
	return nil, nil
}

// limitValue returns the value capped between `maxPositive` and `maxNegative`, alongside a boolean indicating if limits needed to be applied
func limitValue(value, maxPositive, maxNegative float64) (float64, bool) {
	if value > maxPositive {
		return maxPositive, true
	} else if value < -maxNegative {
		return -maxNegative, true
	} else {
		return value, false
	}
}

// sendIfNonBlocking attempts to send the given value onto the given channel, but will only do so if the operation
// is non-blocking, otherwise it logs a warning message and returns.
func sendIfNonBlocking[V any](ch chan<- V, val V, messageTargetLogStr string) {
	select {
	case ch <- val:
	default:
		slog.Warn("Dropped message", "message_target", messageTargetLogStr)
	}
}

// PeriodicalConfigTypes is an interface onto configuration structures that are tied to a particular periods of time
type PeriodicalConfigTypes interface {
	config.ImportAvoidanceWhenShortConfig | config.DayedPeriodWithSoe | config.DayedPeriodWithNIV | config.DynamicPeakDischargeConfig
	GetDayedPeriod() timeutils.DayedPeriod
}

// findPeriodicalConfigForTime searches the list of configs and returns the first one that is active at the time `t` (i.e.
// the first one whose period contains `t`), or nil if none was found. It also returns the associated absolute period.
func findPeriodicalConfigForTime[T PeriodicalConfigTypes](t time.Time, configs []T) (*T, timeutils.Period) {

	for _, conf := range configs {
		dayedPeriod := conf.GetDayedPeriod()
		absPeriod, ok := dayedPeriod.AbsolutePeriod(t)
		if ok {
			return &conf, absPeriod
		}
	}
	return nil, timeutils.Period{}
}

// pointerToFloat64 returns a pointer to the given value. In Go you cannot directly access the address
// of a const value, so you can't write `&5.0` for instance. This function just allows us to get the address.
func pointerToFloat64(val float64) *float64 {
	return &val
}
