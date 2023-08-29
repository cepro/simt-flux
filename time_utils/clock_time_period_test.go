package timeutils

import (
	"testing"
	"time"
)

func TestContains(t *testing.T) {

	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Errorf("Failed to load London time: %v", err)
	}
	sixTo10Am := ClockTimePeriod{
		Start: ClockTime{
			Hour:     6,
			Minute:   0,
			Second:   0,
			Location: london,
		},
		End: ClockTime{
			Hour:     10,
			Minute:   0,
			Second:   0,
			Location: london,
		},
	}

	type subTest struct {
		name     string
		period   ClockTimePeriod
		t        time.Time
		expected bool
	}

	subTests := []subTest{
		{"OutsideBefore", sixTo10Am, time.Date(2023, 8, 22, 0, 0, 0, 0, london), false},
		{"OutsideAfter", sixTo10Am, time.Date(2023, 8, 22, 11, 0, 0, 0, london), false},
		{"ContainsOnStartBoundary", sixTo10Am, time.Date(2023, 8, 22, 6, 0, 0, 0, london), true},
		{"ContainsOnEndBoundary", sixTo10Am, time.Date(2023, 8, 22, 10, 0, 0, 0, london), true},
		{"ContainsInside", sixTo10Am, time.Date(2023, 8, 22, 9, 40, 0, 0, london), true},
	}
	for _, subTest := range subTests {
		t.Run(subTest.name, func(t *testing.T) {
			contains := subTest.period.Contains(subTest.t)
			if contains != subTest.expected {
				t.Errorf("got %t, expected %t", contains, subTest.expected)
			}
		})
	}
}
