package axle

import (
	"time"

	timeutils "github.com/cepro/besscontroller/time_utils"
)

type Schedule struct {
	// TODO: implement schedule based on what Axle send us
	ReceivedTime time.Time
	Actions      []ScheduleAction
}

type ScheduleAction struct {
	Period         timeutils.Period
	ActionType     string
	AllowDeviation bool
}

// FirstActionAt returns the action that is active at time `t`, or nil if there is none.
// If there are multiple actions that are active at `t` then only the first is returned.
func (s *Schedule) FirstActionAt(t time.Time) *ScheduleAction {
	for _, action := range s.Actions {
		if action.Period.Contains(t) {
			return &action
		}
	}
	return nil
}

// Equal checks if the two schedules are equal
func (s *Schedule) Equal(other Schedule, checkRxTime bool) bool {

	if checkRxTime {
		if !s.ReceivedTime.Equal(other.ReceivedTime) {
			return false
		}
	}

	// Compare Actions slice length
	if len(s.Actions) != len(other.Actions) {
		return false
	}

	// Compare each action in the slice
	for i, action := range s.Actions {
		if !action.Equal(other.Actions[i]) {
			return false
		}
	}

	return true
}

// Equal checks if two ScheduleAction instances are equal
func (a *ScheduleAction) Equal(other ScheduleAction) bool {

	return a.Period.Equal(other.Period) &&
		a.ActionType == other.ActionType &&
		a.AllowDeviation == other.AllowDeviation
}
