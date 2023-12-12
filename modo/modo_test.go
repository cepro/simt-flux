package modo

import (
	"testing"
	"time"
)

func TestTimeOfSettlementPeriod(t *testing.T) {

	type subTest struct {
		name         string
		dateStr      string
		sp           int
		expectedTime time.Time
		expectedErr  error
	}

	subTests := []subTest{
		{"GMT1", "2023-12-11", 22, mustParseTime("2023-12-11T10:30:00+00:00"), nil},
		{"GMT2", "2023-12-11", 1, mustParseTime("2023-12-11T00:00:00+00:00"), nil},
		{"GMT3", "2023-12-11", 48, mustParseTime("2023-12-11T23:30:00+00:00"), nil},
		{"BST1", "2023-06-01", 22, mustParseTime("2023-06-01T10:30:00+01:00"), nil},
		{"BST2", "2023-06-01", 3, mustParseTime("2023-06-01T01:00:00+01:00"), nil},
		{"Clock change back 1", "2023-10-29", 1, mustParseTime("2023-10-29T00:00:00+01:00"), nil},
		{"Clock change back 2", "2023-10-29", 3, mustParseTime("2023-10-29T01:00:00+01:00"), nil},
		{"Clock change back 3", "2023-10-29", 4, mustParseTime("2023-10-29T01:30:00+01:00"), nil},
		{"Clock change back 4", "2023-10-29", 5, mustParseTime("2023-10-29T01:00:00+00:00"), nil},
		{"Clock change back 5", "2023-10-29", 6, mustParseTime("2023-10-29T01:30:00+00:00"), nil},
		{"Clock change back 6", "2023-10-29", 7, mustParseTime("2023-10-29T02:00:00+00:00"), nil},
		{"Clock change back 7", "2023-10-29", 50, mustParseTime("2023-10-29T23:30:00+00:00"), nil},
		{"Clock change forward 1", "2023-03-26", 1, mustParseTime("2023-03-26T00:00:00+00:00"), nil},
		{"Clock change forward 2", "2023-03-26", 2, mustParseTime("2023-03-26T00:30:00+00:00"), nil},
		{"Clock change forward 3", "2023-03-26", 3, mustParseTime("2023-03-26T02:00:00+01:00"), nil},
		{"Clock change forward 4", "2023-03-26", 46, mustParseTime("2023-03-26T23:30:00+01:00"), nil},
	}
	for _, subTest := range subTests {
		t.Run(subTest.name, func(t *testing.T) {
			actualTime, err := timeOfSettlementPeriod(subTest.dateStr, subTest.sp)
			if err != subTest.expectedErr {
				t.Errorf("Got error %v, expected error %v", err, subTest.expectedErr)
			}
			if !actualTime.Equal(subTest.expectedTime) {
				t.Errorf("Got %v, expected %v", actualTime, subTest.expectedTime)
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
