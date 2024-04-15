package timeutils

import (
	"testing"
	"time"
)

func TestClockTimeAbsolutePeriod(t *testing.T) {
	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Errorf("Failed to load London time: %v", err)
	}
	london2, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Errorf("Failed to load London time: %v", err)
	}

	sixToTenAm := ClockTimePeriod{
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

	sixToTenAmTwoLocationInstances := ClockTimePeriod{
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
			Location: london2,
		},
	}

	midnightTo3Am := ClockTimePeriod{
		Start: ClockTime{
			Hour:     0,
			Minute:   0,
			Second:   0,
			Location: london,
		},
		End: ClockTime{
			Hour:     3,
			Minute:   0,
			Second:   0,
			Location: london,
		},
	}

	// An 'absolute' version of the sixToTenAm 'clock time period' that occurs on the 22nd of August 2023
	sixTo10AmAbsolute := Period{
		Start: time.Date(2023, 8, 22, 6, 0, 0, 0, london),
		End:   time.Date(2023, 8, 22, 10, 0, 0, 0, london),
	}

	// An 'absolute' version of the midnightTo3Am 'clock time period' that occurs on the 14th of April 2023
	midnightTo3AmAbsolute := Period{
		Start: time.Date(2023, 4, 14, 0, 0, 0, 0, london),
		End:   time.Date(2023, 4, 14, 3, 0, 0, 0, london),
	}

	type subTest struct {
		name           string
		ctPeriod       ClockTimePeriod
		t              time.Time
		expectedPeriod Period
		expectedOK     bool
	}

	subTests := []subTest{
		{"OutsideBefore", sixToTenAm, time.Date(2023, 8, 22, 0, 0, 0, 0, london), Period{}, false},
		{"OutsideAfter", sixToTenAm, time.Date(2023, 8, 22, 11, 0, 0, 0, london), Period{}, false},
		{"ContainsOnStartBoundary", sixToTenAm, time.Date(2023, 8, 22, 6, 0, 0, 0, london), sixTo10AmAbsolute, true},
		{"ContainsOnEndBoundary", sixToTenAm, time.Date(2023, 8, 22, 10, 0, 0, 0, london), Period{}, false},
		{"ContainsInside", sixToTenAm, time.Date(2023, 8, 22, 9, 40, 0, 0, london), sixTo10AmAbsolute, true},

		{"ContainsInside, two location instances", sixToTenAmTwoLocationInstances, time.Date(2023, 8, 22, 9, 40, 0, 0, london), sixTo10AmAbsolute, true},

		{"UTC time input, BST period, before midnight, outside period", midnightTo3Am, time.Date(2023, 04, 13, 22, 59, 0, 0, time.UTC), Period{}, false},
		{"UTC time input, BST period, near midnight, inside period", midnightTo3Am, time.Date(2023, 04, 13, 23, 0, 0, 0, time.UTC), midnightTo3AmAbsolute, true},
		{"UTC time input, BST period, on midnight, inside period", midnightTo3Am, time.Date(2023, 04, 14, 0, 0, 0, 0, time.UTC), midnightTo3AmAbsolute, true},
		{"UTC time input, BST period, after midnight, inside period", midnightTo3Am, time.Date(2023, 04, 14, 1, 30, 0, 0, time.UTC), midnightTo3AmAbsolute, true},
		{"UTC time input, BST period, after midnight, outside period", midnightTo3Am, time.Date(2023, 04, 14, 2, 0, 0, 0, time.UTC), Period{}, false},
	}
	for _, subTest := range subTests {
		t.Run(subTest.name, func(t *testing.T) {
			period, ok := subTest.ctPeriod.AbsolutePeriod(subTest.t)
			if ok != subTest.expectedOK {
				t.Errorf("OK boolean got %t, expected %t", ok, subTest.expectedOK)
			}
			if ok && period.Equal(subTest.expectedPeriod) {
				t.Errorf("Period got %v, expected %v", period, subTest.expectedPeriod)
			}
		})
	}

}
