package timeutils

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ClockTime represents a time of day in the given locale, without a date.
type ClockTime struct {
	Hour     int
	Minute   int
	Second   int
	Location *time.Location
}

func (c *ClockTime) UnmarshalYAML(unmarshal func(interface{}) error) error {

	var str string
	err := unmarshal(&str)
	if err != nil {
		return fmt.Errorf("to string: %w", err)
	}

	elements := strings.Split(str, ":")
	if len(elements) != 4 {
		return fmt.Errorf("ClockTime '%s' expected 4 elements, found %d", str, len(elements))
	}

	hour, err := strconv.Atoi(elements[0])
	if err != nil {
		return err
	}
	minute, err := strconv.Atoi(elements[1])
	if err != nil {
		return err
	}
	second, err := strconv.Atoi(elements[2])
	if err != nil {
		return err
	}
	location, err := time.LoadLocation(elements[3])
	if err != nil {
		return err
	}
	c.Hour = hour
	c.Minute = minute
	c.Second = second
	c.Location = location

	return nil
}

// OnDate returns a time with the given clock time on the given date
func (c *ClockTime) OnDate(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, c.Hour, c.Minute, c.Second, 0, c.Location)
}
