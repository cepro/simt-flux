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

	// An 'absolute' version of the above 'clock time period' that occurs on the 22nd of August 2023
	sixTo10AmAbsolute := Period{
		Start: time.Date(2023, 8, 22, 6, 0, 0, 0, london),
		End:   time.Date(2023, 8, 22, 10, 0, 0, 0, london),
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
		{"ContainsOnEndBoundary", sixToTenAm, time.Date(2023, 8, 22, 10, 0, 0, 0, london), sixTo10AmAbsolute, false},
		{"ContainsInside", sixToTenAm, time.Date(2023, 8, 22, 9, 40, 0, 0, london), sixTo10AmAbsolute, true},
	}
	for _, subTest := range subTests {
		t.Run(subTest.name, func(t *testing.T) {
			period, ok := subTest.ctPeriod.AbsolutePeriod(subTest.t)
			if ok != subTest.expectedOK {
				t.Errorf("OK boolean got %t, expected %t", ok, subTest.expectedOK)
			}
			if ok && period != subTest.expectedPeriod {
				t.Errorf("Period got %t, expected %t", ok, subTest.expectedOK)
			}
		})
	}

}
