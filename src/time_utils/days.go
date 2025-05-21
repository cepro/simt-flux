package timeutils

import (
	"fmt"
	"strings"
	"time"
)

// These constants define the names of days and are used within the `Days` struct.
const (
	WeekendDaysName = "weekends"
	WeekdayDaysName = "weekdays"
	AllDaysName     = "all"
)

// Days specifies which days to apply some configuration to.
type Days struct {
	Name     string         // A string representation of the days, e.g. "weekends", "weekdays", or "all"
	Location *time.Location // We always need a timezone to use day information, e.g. the time instant "2024-04-06T23:30:00Z" is a Friday in UTC, but a Saturday in BST
}

// IsOnDay returns true if the given time is on one of the days that is specified by `d`.
func (d *Days) IsOnDay(t time.Time) bool {

	// Make sure that `t` is in the relevant timezone for the day configuration.
	t = t.In(d.Location)

	switch d.Name {
	case AllDaysName:
		return true // the day is always okay
	case WeekdayDaysName:
		if IsWeekday(t) {
			return true
		} else {
			return false
		}
	case WeekendDaysName:
		if !IsWeekday(t) {
			return true
		} else {
			return false
		}
	default:
		panic(fmt.Sprintf("Unknown day specification: '%s'", d.Name))
	}
}

// UnmarshalYAML defines how a string is converted into a Days struct. A colon is used to delimit the days names from the timezone location
// for example, "weekdays:Europe/London".
func (d *Days) UnmarshalYAML(unmarshal func(interface{}) error) error {

	var str string
	err := unmarshal(&str)
	if err != nil {
		return fmt.Errorf("to string: %w", err)
	}

	elements := strings.Split(str, ":")
	if len(elements) != 2 {
		return fmt.Errorf("Days '%s' expected 2 elements, found %d", str, len(elements))
	}

	d.Name = elements[0]

	location, err := time.LoadLocation(elements[1])
	if err != nil {
		return err
	}

	d.Location = location

	return nil
}
