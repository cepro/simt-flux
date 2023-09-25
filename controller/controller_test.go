package controller

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cepro/besscontroller/telemetry"
	timeutils "github.com/cepro/besscontroller/time_utils"
)

// TestImportAvoidance is a high level test of the controllers ability to issue BessCommands to avoid importing power.
// It feeds the controller with a pre-defined/static timeseries of consumer demand to ensure that it reacts correctly to changes in demand.
func TestImportAvoidance(test *testing.T) {

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
	}

	bessCommands := make(chan telemetry.BessCommand)
	ctrlTickerChan := make(chan time.Time)
	ctrl := New(Config{
		BessNameplatePower:     100,
		BessNameplateEnergy:    200,
		BessMinimumSoE:         200 * 0.05,
		ImportAvoidancePeriods: importAvoidancePeriods,
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

		// A period of increasing demand - the controller should use the battery to match the demand
		{time: mustParseTime("2023-09-12T09:00:04+01:00"), bessSoe: 150, consumerDemand: 25, expectedBessTargetPower: 25},
		{time: mustParseTime("2023-09-12T09:00:05+01:00"), bessSoe: 149, consumerDemand: 50, expectedBessTargetPower: 50},
		{time: mustParseTime("2023-09-12T09:00:06+01:00"), bessSoe: 147, consumerDemand: 75, expectedBessTargetPower: 75},
		{time: mustParseTime("2023-09-12T09:00:07+01:00"), bessSoe: 145, consumerDemand: 100, expectedBessTargetPower: 100},

		// A period where the demand exceeds the batteries power capability - the controller should stick to the maximum power of the battery
		{time: mustParseTime("2023-09-12T09:00:08+01:00"), bessSoe: 143, consumerDemand: 110, expectedBessTargetPower: 100},
		{time: mustParseTime("2023-09-12T09:00:09+01:00"), bessSoe: 141, consumerDemand: 120, expectedBessTargetPower: 100},
		{time: mustParseTime("2023-09-12T09:00:10+01:00"), bessSoe: 139, consumerDemand: 101, expectedBessTargetPower: 100},

		// A period of decreasing demand - the controller should back off to match the demand
		{time: mustParseTime("2023-09-12T09:00:11+01:00"), bessSoe: 144, consumerDemand: 50, expectedBessTargetPower: 50},
		{time: mustParseTime("2023-09-12T09:00:12+01:00"), bessSoe: 142, consumerDemand: 25, expectedBessTargetPower: 25},

		// Another period of zero demand
		{time: mustParseTime("2023-09-12T09:00:13+01:00"), bessSoe: 141, consumerDemand: 0, expectedBessTargetPower: 0},
		{time: mustParseTime("2023-09-12T09:00:14+01:00"), bessSoe: 141, consumerDemand: 0, expectedBessTargetPower: 0},
		{time: mustParseTime("2023-09-12T09:00:15+01:00"), bessSoe: 141, consumerDemand: 0, expectedBessTargetPower: 0},

		// A period of solar surplus - the controller should allow this to be exported
		{time: mustParseTime("2023-09-12T09:00:16+01:00"), bessSoe: 141, consumerDemand: -10, expectedBessTargetPower: 0},
		{time: mustParseTime("2023-09-12T09:00:17+01:00"), bessSoe: 142, consumerDemand: -10, expectedBessTargetPower: 0},

		// Skip to a time where we are outside of any 'import avoidance periods' and import power, the controller should allow the import.
		// Currently a discontinuity in time is not an issue for the controller, but might be better to split this into a seperate test long-term.
		{time: mustParseTime("2023-09-12T10:00:01+01:00"), bessSoe: 142, consumerDemand: 0, expectedBessTargetPower: 0},
		{time: mustParseTime("2023-09-12T10:00:02+01:00"), bessSoe: 142, consumerDemand: 10, expectedBessTargetPower: 0},
		{time: mustParseTime("2023-09-12T10:00:03+01:00"), bessSoe: 142, consumerDemand: 10, expectedBessTargetPower: 0},
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

		if mock.bessTargetPower != point.expectedBessTargetPower {
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
		PowerTotalActive: sitePower,
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
