package timeutils

import (
	"testing"
	"time"
)

func TestFloorHH(t *testing.T) {

	type subTest struct {
		name      string
		t         time.Time
		expectedT time.Time
	}

	subTests := []subTest{
		{"BST-1", mustParseTime("2023-09-12T09:00:00+01:00"), mustParseTime("2023-09-12T09:00:00+01:00")},
		{"BST-2", mustParseTime("2023-09-12T09:10:00+01:00"), mustParseTime("2023-09-12T09:00:00+01:00")},
		{"BST-3", mustParseTime("2023-09-12T09:29:29+01:00"), mustParseTime("2023-09-12T09:00:00+01:00")},
		{"BST-4", mustParseTime("2023-09-12T09:30:00+01:00"), mustParseTime("2023-09-12T09:30:00+01:00")},
		{"BST-5", mustParseTime("2023-09-12T09:40:00+01:00"), mustParseTime("2023-09-12T09:30:00+01:00")},
		{"BST-6", mustParseTime("2023-09-12T09:59:59+01:00"), mustParseTime("2023-09-12T09:30:00+01:00")},
		{"GMT-1", mustParseTime("2023-11-01T09:59:59+00:00"), mustParseTime("2023-11-01T09:30:00+00:00")},
	}
	for _, subTest := range subTests {
		t.Run(subTest.name, func(t *testing.T) {
			actualT := FloorHH(subTest.t)
			if actualT != subTest.expectedT {
				t.Errorf("Got %v, expected %v", actualT, subTest.expectedT)
			}
		})
	}

}

// mustParseTime returns the time.Time associated with the given string or panics.
func mustParseTime(str string) time.Time {
	time, err := time.Parse(time.RFC3339, str)
	if err != nil {
		panic(err)
	}
	return time
}
