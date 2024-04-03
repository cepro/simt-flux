package timeutils

import (
	"testing"
	"time"
)

func TestIsOnDay(t *testing.T) {
	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Errorf("Failed to load London time: %v", err)
	}

	// the actual time of day is arbitrary for this test
	ctPeriod := ClockTimePeriod{
		Start: ClockTime{
			Hour:     17,
			Minute:   0,
			Second:   0,
			Location: london,
		},
		End: ClockTime{
			Hour:     19,
			Minute:   0,
			Second:   0,
			Location: london,
		},
	}

	weekdaysPeriod := DayedPeriod{
		Days:            WeekdayDays,
		ClockTimePeriod: ctPeriod,
	}

	weekendsPeriod := DayedPeriod{
		Days:            WeekendDays,
		ClockTimePeriod: ctPeriod,
	}

	allDaysPeriod := DayedPeriod{
		Days:            AllDays,
		ClockTimePeriod: ctPeriod,
	}

	type subTest struct {
		name            string
		dayedPeriod     DayedPeriod
		t               time.Time
		expectedIsOnDay bool
	}

	subTests := []subTest{
		{"WeekdayMatchMonday", weekdaysPeriod, time.Date(2024, 4, 1, 18, 0, 0, 0, london), true},
		{"WeekdayMatchFriday", weekdaysPeriod, time.Date(2024, 4, 5, 18, 0, 0, 0, london), true},
		{"WeekdayNoMatchSaturday", weekdaysPeriod, time.Date(2024, 4, 6, 18, 0, 0, 0, london), false},
		{"WeekdayNoMatchSunday", weekdaysPeriod, time.Date(2024, 4, 7, 18, 0, 0, 0, london), false},

		{"WeekendNoMatchMonday", weekendsPeriod, time.Date(2024, 4, 1, 18, 0, 0, 0, london), false},
		{"WeekendNoMatchFriday", weekendsPeriod, time.Date(2024, 4, 5, 18, 0, 0, 0, london), false},
		{"WeekendMatchSaturday", weekendsPeriod, time.Date(2024, 4, 6, 18, 0, 0, 0, london), true},
		{"WeekendMatchSunday", weekendsPeriod, time.Date(2024, 4, 7, 18, 0, 0, 0, london), true},

		{"AllDaysMatchMonday", allDaysPeriod, time.Date(2024, 4, 1, 18, 0, 0, 0, london), true},
		{"AllDaysMatchFriday", allDaysPeriod, time.Date(2024, 4, 5, 18, 0, 0, 0, london), true},
		{"AllDaysMatchSaturday", allDaysPeriod, time.Date(2024, 4, 6, 18, 0, 0, 0, london), true},
		{"AllDaysMatchSunday", allDaysPeriod, time.Date(2024, 4, 7, 18, 0, 0, 0, london), true},
	}

	for _, subTest := range subTests {
		t.Run(subTest.name, func(t *testing.T) {
			isOnDay := subTest.dayedPeriod.IsOnDay(subTest.t)
			if isOnDay != subTest.expectedIsOnDay {
				t.Errorf("IsOnDay boolean got %t, expected %t", isOnDay, subTest.expectedIsOnDay)
			}
		})
	}

}
