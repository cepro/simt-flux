package controller

import "time"

// timedMetric is a float64 value that has an associated time at which it was last updated.
type timedMetric struct {
	value     float64
	updatedAt time.Time
}

// set updates the value and time of the metric
func (t *timedMetric) set(value float64) {
	t.value = value
	t.updatedAt = time.Now()
}

// isOlderThan returns true if the metric's value is older than the given age
func (t *timedMetric) isOlderThan(age time.Duration) bool {
	return time.Now().Sub(t.updatedAt) > age
}
