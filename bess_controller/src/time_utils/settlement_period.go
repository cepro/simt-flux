package timeutils

import "time"

const (
	ThirtyMins = time.Minute * 30
)

// DurationLeftOfSP returns the amount of time remaining in the settlement period, given the current time `t`.
func DurationLeftOfSP(t time.Time) time.Duration {
	spStart := FloorHH(t)
	durationLeft := ThirtyMins - t.Sub(spStart)
	return durationLeft
}
