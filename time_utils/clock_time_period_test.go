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

func TestNextStartTimes(t *testing.T) {

	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Errorf("Failed to load London time: %v", err)
	}

	periods := []ClockTimePeriod{
		{
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
		},
		{
			Start: ClockTime{
				Hour:     16,
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
		},
	}

	type subTest struct {
		name     string
		periods  []ClockTimePeriod
		t        time.Time
		expected []time.Time
	}

	subTests := []subTest{
		{
			name:    "Before all periods",
			periods: periods,
			t:       time.Date(2023, 8, 22, 4, 0, 0, 0, london),
			expected: []time.Time{
				time.Date(2023, 8, 22, 6, 0, 0, 0, london),
				time.Date(2023, 8, 22, 16, 0, 0, 0, london),
			},
		},
		{
			name:    "Within the first period",
			periods: periods,
			t:       time.Date(2023, 8, 22, 8, 0, 0, 0, london),
			expected: []time.Time{
				time.Date(2023, 8, 22, 16, 0, 0, 0, london),
				time.Date(2023, 8, 23, 6, 0, 0, 0, london),
			},
		},
		{
			name:    "Between the first and second periods",
			periods: periods,
			t:       time.Date(2023, 8, 22, 13, 0, 0, 0, london),
			expected: []time.Time{
				time.Date(2023, 8, 22, 16, 0, 0, 0, london),
				time.Date(2023, 8, 23, 6, 0, 0, 0, london),
			},
		},
		{
			name:    "Within the second period",
			periods: periods,
			t:       time.Date(2023, 8, 22, 19, 30, 0, 0, london),
			expected: []time.Time{
				time.Date(2023, 8, 23, 6, 0, 0, 0, london),
				time.Date(2023, 8, 23, 16, 0, 0, 0, london),
			},
		},
		{
			name:    "After the second period",
			periods: periods,
			t:       time.Date(2023, 8, 22, 23, 30, 0, 0, london),
			expected: []time.Time{
				time.Date(2023, 8, 23, 6, 0, 0, 0, london),
				time.Date(2023, 8, 23, 16, 0, 0, 0, london),
			},
		},
	}
	for _, subTest := range subTests {
		t.Run(subTest.name, func(t *testing.T) {
			startTimes := NextStartTimes(subTest.t, subTest.periods)
			if len(startTimes) != len(subTest.expected) {
				t.Errorf("Length of got '%d', doesn't match expected '%d'", len(startTimes), len(subTest.expected))
				return
			}

			for i := range startTimes {
				if startTimes[i] != subTest.expected[i] {
					t.Errorf("At index %d got '%s', expected '%s'", i, startTimes[i].Format(time.RFC3339), subTest.expected[i].Format(time.RFC3339))
				}
			}
		})
	}
}
