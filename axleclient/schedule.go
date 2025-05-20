package axleclient

import (
	"time"

	timeutils "github.com/cepro/besscontroller/time_utils"
)

type Schedule struct {
	ReceivedTime time.Time
	Items        []ScheduleItem `json:"schedule_steps"`
}

type ScheduleItem struct {
	Start          time.Time `json:"start_timestamp"`
	End            time.Time `json:"end_timestamp"`
	Action         string    `json:"action"`
	AllowDeviation bool      `json:"allow_deviation"`
}

// FirstItemAt returns the item that is active at time `t`, or nil if there is none.
// If there are multiple items that are active at `t` then only the first is returned.
func (s *Schedule) FirstItemAt(t time.Time) *ScheduleItem {
	for _, item := range s.Items {
		if item.Period().Contains(t) {
			return &item
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
	if len(s.Items) != len(other.Items) {
		return false
	}

	// Compare each action in the slice
	for i, action := range s.Items {
		if !action.Equal(other.Items[i]) {
			return false
		}
	}

	return true
}

// Equal checks if two ScheduleItem instances are equal
func (i *ScheduleItem) Equal(other ScheduleItem) bool {

	return i.Period().Equal(other.Period()) &&
		i.Action == other.Action &&
		i.AllowDeviation == other.AllowDeviation
}

func (i *ScheduleItem) Period() timeutils.Period {
	return timeutils.Period{
		Start: i.Start,
		End:   i.End,
	}
}
