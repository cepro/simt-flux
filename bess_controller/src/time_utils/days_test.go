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

	weekdaysLondon := Days{
		Name:     WeekdayDaysName,
		Location: london,
	}

	weekendsLondon := Days{
		Name:     WeekendDaysName,
		Location: london,
	}

	alldaysLondon := Days{
		Name:     AllDaysName,
		Location: london,
	}

	type subTest struct {
		name            string
		days            Days
		t               time.Time
		expectedIsOnDay bool
	}

	subTests := []subTest{
		{"WeekdayMatchMonday", weekdaysLondon, time.Date(2024, 4, 1, 18, 0, 0, 0, london), true},
		{"WeekdayMatchFriday", weekdaysLondon, time.Date(2024, 4, 5, 18, 0, 0, 0, london), true},
		{"WeekdayNoMatchSaturday", weekdaysLondon, time.Date(2024, 4, 6, 18, 0, 0, 0, london), false},
		{"WeekdayNoMatchSunday", weekdaysLondon, time.Date(2024, 4, 7, 18, 0, 0, 0, london), false},

		{"WeekendNoMatchMonday", weekendsLondon, time.Date(2024, 4, 1, 18, 0, 0, 0, london), false},
		{"WeekendNoMatchFriday", weekendsLondon, time.Date(2024, 4, 5, 18, 0, 0, 0, london), false},
		{"WeekendMatchSaturday", weekendsLondon, time.Date(2024, 4, 6, 18, 0, 0, 0, london), true},
		{"WeekendMatchSunday", weekendsLondon, time.Date(2024, 4, 7, 18, 0, 0, 0, london), true},

		{"AllDaysMatchMonday", alldaysLondon, time.Date(2024, 4, 1, 18, 0, 0, 0, london), true},
		{"AllDaysMatchFriday", alldaysLondon, time.Date(2024, 4, 5, 18, 0, 0, 0, london), true},
		{"AllDaysMatchSaturday", alldaysLondon, time.Date(2024, 4, 6, 18, 0, 0, 0, london), true},
		{"AllDaysMatchSunday", alldaysLondon, time.Date(2024, 4, 7, 18, 0, 0, 0, london), true},

		{"WeekendMatchSaturday UTC to BST", weekendsLondon, time.Date(2024, 4, 5, 23, 00, 0, 0, time.UTC), true}, // The time is given in UTC, but needs to be converted to BST for accurate day calculations
	}

	for _, subTest := range subTests {
		t.Run(subTest.name, func(t *testing.T) {
			isOnDay := subTest.days.IsOnDay(subTest.t)
			if isOnDay != subTest.expectedIsOnDay {
				t.Errorf("IsOnDay boolean got %t, expected %t", isOnDay, subTest.expectedIsOnDay)
			}
		})
	}

}
