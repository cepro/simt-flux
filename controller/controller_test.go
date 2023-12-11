package controller

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/cepro/besscontroller/config"
	"github.com/cepro/besscontroller/telemetry"
	timeutils "github.com/cepro/besscontroller/time_utils"
)

const (
	chargeEfficiency = 0.9
)

// TestController is a high level test of the controllers ability to issue BessCommands to service the "import avoidance", "export avoidance" and "charge to min" modes.
// It feeds the controller with a pre-defined/static timeseries of consumer demand to ensure that it reacts correctly to changes in demand.
func TestController(test *testing.T) {

	ctx := context.Background()

	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		test.Fatalf("Could not load location: %v", err)
	}
	importAvoidancePeriods := []timeutils.ClockTimePeriod{
		{
			Start: timeutils.ClockTime{Hour: 9, Minute: 0, Second: 0, Location: london},
			End:   timeutils.ClockTime{Hour: 10, Minute: 0, Second: 0, Location: london},
		},
		{
			Start: timeutils.ClockTime{Hour: 15, Minute: 0, Second: 0, Location: london},
			End:   timeutils.ClockTime{Hour: 16, Minute: 0, Second: 0, Location: london},
		},
		{
			Start: timeutils.ClockTime{Hour: 21, Minute: 0, Second: 0, Location: london},
			End:   timeutils.ClockTime{Hour: 22, Minute: 0, Second: 0, Location: london},
		},
	}
	exportAvoidancePeriods := []timeutils.ClockTimePeriod{
		{
			Start: timeutils.ClockTime{Hour: 11, Minute: 0, Second: 0, Location: london},
			End:   timeutils.ClockTime{Hour: 12, Minute: 0, Second: 0, Location: london},
		},
		{
			Start: timeutils.ClockTime{Hour: 15, Minute: 0, Second: 0, Location: london},
			End:   timeutils.ClockTime{Hour: 16, Minute: 0, Second: 0, Location: london},
		},
		{
			Start: timeutils.ClockTime{Hour: 17, Minute: 0, Second: 0, Location: london},
			End:   timeutils.ClockTime{Hour: 18, Minute: 0, Second: 0, Location: london},
		},
		{
			Start: timeutils.ClockTime{Hour: 21, Minute: 0, Second: 0, Location: london},
			End:   timeutils.ClockTime{Hour: 22, Minute: 0, Second: 0, Location: london},
		},
	}
	chargeToMinPeriods := []config.ClockTimePeriodWithSoe{
		{
			Soe: 160,
			Period: timeutils.ClockTimePeriod{
				Start: timeutils.ClockTime{Hour: 13, Minute: 0, Second: 0, Location: london},
				End:   timeutils.ClockTime{Hour: 14, Minute: 0, Second: 0, Location: london},
			},
		},
		{
			Soe: 190,
			Period: timeutils.ClockTimePeriod{
				Start: timeutils.ClockTime{Hour: 17, Minute: 0, Second: 0, Location: london},
				End:   timeutils.ClockTime{Hour: 18, Minute: 0, Second: 0, Location: london},
			},
		},
	}

	bessCommands := make(chan telemetry.BessCommand, 1)
	ctrlTickerChan := make(chan time.Time, 1)
	ctrl := New(Config{
		BessNameplatePower:     100,
		BessNameplateEnergy:    200,
		BessSoeMin:             20,
		BessSoeMax:             180,
		ImportAvoidancePeriods: importAvoidancePeriods,
		ExportAvoidancePeriods: exportAvoidancePeriods,
		BessChargeEfficiency:   chargeEfficiency,
		ChargeToMinPeriods:     chargeToMinPeriods,
		MaxReadingAge:          5 * time.Second,
		BessCommands:           bessCommands,
	})
	go ctrl.Run(ctx, ctrlTickerChan)

	// testpoint represents a point in time that we are testing as part of a larger timeseries
	type testpoint struct {
		time                    time.Time // the point in time being tested
		bessSoe                 float64   // the state of energy of the battery at this point in time
		consumerDemand          float64   // the consumer demand at this point in time
		expectedBessTargetPower float64   // the power command that we expect the controller to issue at this point in time
	}
	testpoints := []testpoint{
		// Start off with zero consumer demand, expecting zero power from the battery
		{time: mustParseTime("2023-09-12T09:00:00+01:00"), bessSoe: 150, consumerDemand: 0, expectedBessTargetPower: 0},
		{time: mustParseTime("2023-09-12T09:00:01+01:00"), bessSoe: 150, consumerDemand: 0, expectedBessTargetPower: 0},
		{time: mustParseTime("2023-09-12T09:00:02+01:00"), bessSoe: 150, consumerDemand: 0, expectedBessTargetPower: 0},
		{time: mustParseTime("2023-09-12T09:00:03+01:00"), bessSoe: 150, consumerDemand: 0, expectedBessTargetPower: 0},

		// A period of increasing demand whilst we are in 'import avoidance' - the controller should use the battery to match the demand - which should reduce site import
		{time: mustParseTime("2023-09-12T09:00:04+01:00"), bessSoe: 150, consumerDemand: 25, expectedBessTargetPower: 25},
		{time: mustParseTime("2023-09-12T09:00:05+01:00"), bessSoe: 149, consumerDemand: 50, expectedBessTargetPower: 50},
		{time: mustParseTime("2023-09-12T09:00:06+01:00"), bessSoe: 147, consumerDemand: 75, expectedBessTargetPower: 75},
		{time: mustParseTime("2023-09-12T09:00:07+01:00"), bessSoe: 145, consumerDemand: 100, expectedBessTargetPower: 100},

		// A period where the demand exceeds the batteries power capability whilst we are in 'import avoidance' - the controller should stick to the maximum power of the battery
		{time: mustParseTime("2023-09-12T09:00:08+01:00"), bessSoe: 143, consumerDemand: 110, expectedBessTargetPower: 100},
		{time: mustParseTime("2023-09-12T09:00:09+01:00"), bessSoe: 141, consumerDemand: 120, expectedBessTargetPower: 100},
		{time: mustParseTime("2023-09-12T09:00:10+01:00"), bessSoe: 139, consumerDemand: 101, expectedBessTargetPower: 100},

		// A period of decreasing demand whilst we are in 'import avoidance' - the controller should back off to match the demand
		{time: mustParseTime("2023-09-12T09:00:11+01:00"), bessSoe: 144, consumerDemand: 50, expectedBessTargetPower: 50},
		{time: mustParseTime("2023-09-12T09:00:12+01:00"), bessSoe: 142, consumerDemand: 25, expectedBessTargetPower: 25},

		// Another period of zero demand whilst we are in 'import avoidance'
		{time: mustParseTime("2023-09-12T09:00:13+01:00"), bessSoe: 141, consumerDemand: 0, expectedBessTargetPower: 0},
		{time: mustParseTime("2023-09-12T09:00:14+01:00"), bessSoe: 141, consumerDemand: 0, expectedBessTargetPower: 0},
		{time: mustParseTime("2023-09-12T09:00:15+01:00"), bessSoe: 141, consumerDemand: 0, expectedBessTargetPower: 0},

		// A period of solar surplus whilst we are in 'import avoidance' - the controller should allow this to be exported
		{time: mustParseTime("2023-09-12T09:00:16+01:00"), bessSoe: 141, consumerDemand: -10, expectedBessTargetPower: 0},
		{time: mustParseTime("2023-09-12T09:00:17+01:00"), bessSoe: 142, consumerDemand: -10, expectedBessTargetPower: 0},

		// Currently a discontinuity in time is not an issue for the controller...

		// Skip to a time where we are outside of any configured acivity, the controller should do nothing
		{time: mustParseTime("2023-09-12T10:00:16+01:00"), bessSoe: 140, consumerDemand: -10, expectedBessTargetPower: 0},
		{time: mustParseTime("2023-09-12T10:00:17+01:00"), bessSoe: 140, consumerDemand: -10, expectedBessTargetPower: 0},
		{time: mustParseTime("2023-09-12T10:00:18+01:00"), bessSoe: 140, consumerDemand: 10, expectedBessTargetPower: 0},
		{time: mustParseTime("2023-09-12T10:00:19+01:00"), bessSoe: 140, consumerDemand: 10, expectedBessTargetPower: 0},

		// Skip to a time wher we are in 'export avoidance' - the controller should prevent export
		{time: mustParseTime("2023-09-12T11:00:00+01:00"), bessSoe: 100, consumerDemand: 15, expectedBessTargetPower: 0},
		{time: mustParseTime("2023-09-12T11:00:01+01:00"), bessSoe: 100, consumerDemand: 0, expectedBessTargetPower: 0},
		{time: mustParseTime("2023-09-12T11:00:02+01:00"), bessSoe: 100, consumerDemand: -10, expectedBessTargetPower: -10},
		{time: mustParseTime("2023-09-12T11:00:03+01:00"), bessSoe: 101, consumerDemand: -50, expectedBessTargetPower: -50},
		{time: mustParseTime("2023-09-12T11:00:04+01:00"), bessSoe: 102, consumerDemand: -500, expectedBessTargetPower: -100},
		{time: mustParseTime("2023-09-12T11:00:05+01:00"), bessSoe: 103, consumerDemand: 15, expectedBessTargetPower: 0},

		// Skip to a time wher we are in 'charge to min' - the controller should charge to reach the minimum soe
		{time: mustParseTime("2023-09-12T13:00:00+01:00"), bessSoe: 100, consumerDemand: 15, expectedBessTargetPower: -60 / chargeEfficiency},
		{time: mustParseTime("2023-09-12T13:00:01+01:00"), bessSoe: 100, consumerDemand: 0, expectedBessTargetPower: -60 / chargeEfficiency},
		{time: mustParseTime("2023-09-12T13:00:02+01:00"), bessSoe: 100, consumerDemand: 15, expectedBessTargetPower: -60 / chargeEfficiency},

		// Skip to a time when both 'export avoidance' and 'import avoidance' are active
		{time: mustParseTime("2023-09-12T15:00:00+01:00"), bessSoe: 160, consumerDemand: 15, expectedBessTargetPower: 15},
		{time: mustParseTime("2023-09-12T15:00:01+01:00"), bessSoe: 160, consumerDemand: 0, expectedBessTargetPower: 0},
		{time: mustParseTime("2023-09-12T15:00:02+01:00"), bessSoe: 160, consumerDemand: -15, expectedBessTargetPower: -15},

		// Skip to a time when both 'export avoidance' and 'charge to min' are active - the controller should prioritise 'charge to min'
		{time: mustParseTime("2023-09-12T17:00:00+01:00"), bessSoe: 160, consumerDemand: 15, expectedBessTargetPower: -30 / chargeEfficiency},
		{time: mustParseTime("2023-09-12T17:00:01+01:00"), bessSoe: 160, consumerDemand: 0, expectedBessTargetPower: -30 / chargeEfficiency},
		{time: mustParseTime("2023-09-12T17:00:02+01:00"), bessSoe: 160, consumerDemand: -15, expectedBessTargetPower: -30 / chargeEfficiency},
		{time: mustParseTime("2023-09-12T17:00:03+01:00"), bessSoe: 160, consumerDemand: -100, expectedBessTargetPower: -30 / chargeEfficiency},

		// Ensure that the maximum bess soe is honored (import and export avoidance are active for these test points)
		{time: mustParseTime("2023-09-12T21:00:00+01:00"), bessSoe: 179, consumerDemand: -50, expectedBessTargetPower: -50},
		{time: mustParseTime("2023-09-12T21:00:01+01:00"), bessSoe: 180, consumerDemand: -50, expectedBessTargetPower: 0},
		{time: mustParseTime("2023-09-12T21:00:02+01:00"), bessSoe: 180, consumerDemand: -50, expectedBessTargetPower: 0},
		{time: mustParseTime("2023-09-12T21:00:03+01:00"), bessSoe: 181, consumerDemand: -50, expectedBessTargetPower: 0},
		// Ensure that the minimum bess soe is honored (import and export avoidance are active for these test points)
		{time: mustParseTime("2023-09-12T21:30:00+01:00"), bessSoe: 21, consumerDemand: 10, expectedBessTargetPower: 10},
		{time: mustParseTime("2023-09-12T21:30:01+01:00"), bessSoe: 20, consumerDemand: 10, expectedBessTargetPower: 0},
		{time: mustParseTime("2023-09-12T21:30:02+01:00"), bessSoe: 20, consumerDemand: 10, expectedBessTargetPower: 0},
		{time: mustParseTime("2023-09-12T21:30:03+01:00"), bessSoe: 19, consumerDemand: 10, expectedBessTargetPower: 0},
	}

	mock := microgridMock{
		SiteMeterReadings: ctrl.SiteMeterReadings,
		BessReadings:      ctrl.BessReadings,
		BessCommands:      bessCommands,
	}
	for _, point := range testpoints {
		test.Logf("Simulating time %v", point.time)

		// generate the meter and bess readings, using the mocked consumer demand
		mock.SimulateReadings(point.consumerDemand, point.bessSoe)

		// allow time for the readings to be digested by the controller
		time.Sleep(5 * time.Millisecond)

		// trigger the controller to run a control loop
		ctrlTickerChan <- point.time

		// wait for the controller to issue a command
		err := mock.WaitForBessCommand()
		if err != nil {
			test.Errorf("At time '%v', failed to wait for bess command: %v", point.time, err)
			return
		}

		if !almostEqual(mock.bessTargetPower, point.expectedBessTargetPower, 0.1) {
			test.Errorf("At time '%v' got unexpected bess target power: %f, expected: %f", point.time, mock.bessTargetPower, point.expectedBessTargetPower)
			return
		}

	}

}

// microgridMock acts as a mock meter, BESS and consumer demand to enable testing of the controller.
type microgridMock struct {
	SiteMeterReadings chan<- telemetry.MeterReading // The controller under test can take site meter readings from this channel
	BessReadings      chan<- telemetry.BessReading  // The controller under test can take bess readings from this channel
	BessCommands      <-chan telemetry.BessCommand  // The controller under test can put bess commands onto this channel

	bessTargetPower float64 // the power going into or out of the bess
}

// SimulateReadings generates mocked meter and bess readings and sends them to the controller.
func (m microgridMock) SimulateReadings(consumerDemand float64, bessSoe float64) {
	// Assume that the 'site power' is exactly the consumer demand minus the bess target power for now.
	// Later we might want to add some noise or more real-world behaviour.
	sitePower := consumerDemand - m.bessTargetPower

	m.SiteMeterReadings <- telemetry.MeterReading{
		PowerTotalActive: &sitePower,
	}

	// generate a mock bess reading - currently we always have a static SoE
	m.BessReadings <- telemetry.BessReading{
		Soe: bessSoe,
	}
}

// WaitForBessCommand waits for up to one second for the controller to send a new command for the BESS and then stores the command in memory.
func (m *microgridMock) WaitForBessCommand() error {
	select {
	case command := <-m.BessCommands:
		// The controller has sent us a command, just store the target power in memory for this mock.
		m.bessTargetPower = command.TargetPower
		return nil
	case <-time.After(time.Second):
		return fmt.Errorf("timed out")
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

// almostEqual compares two floats, allowing for the given tolerance
func almostEqual(a, b, tolerance float64) bool {
	diff := math.Abs(a - b)
	return diff < tolerance
}
