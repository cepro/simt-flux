package controller

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/cepro/besscontroller/cartesian"
	"github.com/cepro/besscontroller/config"
	"github.com/cepro/besscontroller/telemetry"
	timeutils "github.com/cepro/besscontroller/time_utils"
)

const (
	chargeEfficiency = 0.9
)

// TestController is a high level (almost integration) test of the controllers ability to issue BessCommands to service various control modes.
// It feeds the controller with a pre-defined/static timeseries of consumer demand and prices to ensure that it reacts correctly.
func TestController(test *testing.T) {

	ctx := context.Background()

	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		test.Fatalf("Could not load location: %v", err)
	}
	weekdayImportAvoidancePeriods := []timeutils.ClockTimePeriod{
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
		{
			Start: timeutils.ClockTime{Hour: 23, Minute: 30, Second: 0, Location: london},
			End:   timeutils.ClockTime{Hour: 23, Minute: 59, Second: 59, Location: london},
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
	chargeToSoePeriods := []config.DayedPeriodWithSoe{
		{
			Soe: 130,
			Period: timeutils.ClockTimePeriod{
				Start: timeutils.ClockTime{Hour: 13, Minute: 0, Second: 0, Location: london},
				End:   timeutils.ClockTime{Hour: 13, Minute: 30, Second: 0, Location: london},
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
	dischargeToSoePeriods := []config.DayedPeriodWithSoe{
		{
			Soe: 70,
			Period: timeutils.ClockTimePeriod{
				Start: timeutils.ClockTime{Hour: 13, Minute: 30, Second: 0, Location: london},
				End:   timeutils.ClockTime{Hour: 14, Minute: 0, Second: 0, Location: london},
			},
		},
	}

	nivChasePeriods := []config.ClockTimePeriodWithNIV{
		{
			Period: timeutils.ClockTimePeriod{
				Start: timeutils.ClockTime{Hour: 23, Minute: 0, Second: 0, Location: london},
				End:   timeutils.ClockTime{Hour: 23, Minute: 59, Second: 59, Location: london},
			},
			Niv: config.NivConfig{
				ChargeCurve: cartesian.Curve{
					Points: []cartesian.Point{
						{X: -9999, Y: 180},
						{X: 0, Y: 180},
						{X: 20, Y: 0},
					},
				},
				DischargeCurve: cartesian.Curve{
					Points: []cartesian.Point{
						{X: 30, Y: 180},
						{X: 40, Y: 0},
						{X: 9999, Y: 0},
					},
				},
				CurveShiftLong:  0,
				CurveShiftShort: 0,
				DefaultPricing:  []config.TimedCharge{},
			},
		},
	}

	chargesPeriods := []timeutils.ClockTimePeriod{nivChasePeriods[0].Period} // This is unrealistic but convenient for the test conciseness
	chargesImport := []config.TimedCharge{
		{
			Rate:           10,
			PeriodsWeekday: chargesPeriods,
			PeriodsWeekend: chargesPeriods,
		},
	}
	chargesExport := []config.TimedCharge{
		{
			Rate:           -10,
			PeriodsWeekday: chargesPeriods,
			PeriodsWeekend: chargesPeriods,
		},
	}

	// These are defined as vars because Go cannot get the address of a constant - e.g. you can't do `&50.0`
	var fifty float64 = 50.0
	var seventy float64 = 70.0

	bessCommands := make(chan telemetry.BessCommand, 1)
	ctrlTickerChan := make(chan time.Time, 1)
	ctrl := New(Config{
		BessChargeEfficiency:          chargeEfficiency,
		BessSoeMin:                    20,
		BessSoeMax:                    180,
		BessChargePowerLimit:          100,
		BessDischargePowerLimit:       105,  // slightly higher discharge limit than charge limit for testing the asymmetry
		SiteImportPowerLimit:          9999, // this is replaced at each test iteration
		SiteExportPowerLimit:          9999, // this is replaced at each test iteration
		WeekdayImportAvoidancePeriods: weekdayImportAvoidancePeriods,
		WeekendImportAvoidancePeriods: []timeutils.ClockTimePeriod{},
		ExportAvoidancePeriods:        exportAvoidancePeriods,
		ChargeToSoePeriods:            chargeToSoePeriods,
		WeekdayDischargeToSoePeriods:  dischargeToSoePeriods,
		NivChasePeriods:               nivChasePeriods,
		ChargesImport:                 chargesImport,
		ChargesExport:                 chargesExport,
		ModoClient:                    &MockImbalancePricer{}, // this is replaced at each test iteration
		MaxReadingAge:                 5 * time.Second,
		BessCommands:                  bessCommands,
	})
	go ctrl.Run(ctx, ctrlTickerChan)

	// testpoint represents a point in time that we are testing as part of a larger timeseries
	type testpoint struct {
		time                    time.Time // the point in time being tested
		bessSoe                 float64   // the state of energy of the battery at this point in time
		consumerDemand          float64   // the consumer demand at this point in time
		imbalancePrice          float64   // the predicted system settelment price for this period
		imbalanceVolume         float64   // the predicted system settelment volume for this period
		siteImportPowerLimit    *float64  // site import limits applied for the test point, defaults to 9999 if nil
		siteExportPowerLimit    *float64  // site export limits applied for the test point, defaults to 9999 if nil
		expectedBessTargetPower float64   // the power command that we expect the controller to issue at this point in time
	}
	// TODO: this array of test points covers many test scenarios, it would be better if this was refactored so that each scenario was kept more separate somehow
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
		{time: mustParseTime("2023-09-12T09:00:08+01:00"), bessSoe: 143, consumerDemand: 110, expectedBessTargetPower: 105},
		{time: mustParseTime("2023-09-12T09:00:09+01:00"), bessSoe: 141, consumerDemand: 120, expectedBessTargetPower: 105},
		{time: mustParseTime("2023-09-12T09:00:10+01:00"), bessSoe: 139, consumerDemand: 106, expectedBessTargetPower: 105},

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

		// Skup to a time when we are in 'import avoidance' but it's at the weekend so it shouldn't apply
		{time: mustParseTime("2023-09-09T09:00:06+01:00"), bessSoe: 147, consumerDemand: 75, expectedBessTargetPower: 0},

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

		// Skip to a time wher we are in 'charge to soe' - the controller should charge to reach the target soe
		{time: mustParseTime("2023-09-12T13:00:00+01:00"), bessSoe: 100, consumerDemand: 15, expectedBessTargetPower: -60 / chargeEfficiency},
		{time: mustParseTime("2023-09-12T13:00:01+01:00"), bessSoe: 100, consumerDemand: 0, expectedBessTargetPower: -60 / chargeEfficiency},
		{time: mustParseTime("2023-09-12T13:00:02+01:00"), bessSoe: 100, consumerDemand: 15, expectedBessTargetPower: -60 / chargeEfficiency},

		// Skip to a time wher we are in 'discharge to soe' - the controller should discharge to reach the target soe
		{time: mustParseTime("2023-09-12T13:30:00+01:00"), bessSoe: 100, consumerDemand: 15, expectedBessTargetPower: 60},
		{time: mustParseTime("2023-09-12T13:30:01+01:00"), bessSoe: 200, consumerDemand: 0, expectedBessTargetPower: 105},
		{time: mustParseTime("2023-09-12T13:30:02+01:00"), bessSoe: 100, consumerDemand: 15, expectedBessTargetPower: 60},

		// Skup to a time when we are in 'discharge to soe' but it's at the weekend so it shouldn't apply
		{time: mustParseTime("2023-09-10T13:30:02+01:00"), bessSoe: 100, consumerDemand: 15, expectedBessTargetPower: 0},

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

		// Test NIV chasing...

		// Imbalance price is very attractive for charge - DUoS charges plus imbalance is 0p/kWh - charge at full rate, but abide by charge limits
		{time: mustParseTime("2023-09-12T23:10:00+01:00"), bessSoe: 0, consumerDemand: 10, imbalancePrice: -10, expectedBessTargetPower: -100},

		// Imbalance price is very attractive for discharge - DUoS charges plus imbalance is 70p/kWh - discharge at full rate, but abide by discharge limits
		{time: mustParseTime("2023-09-12T23:10:00+01:00"), bessSoe: 180, consumerDemand: 10, imbalancePrice: 60, expectedBessTargetPower: 105},

		// Imbalance price is attractive for discharge - DUoS charges plus imbalance is 70p/kWh - but we are limited by site export limits, which also track any load/generation from the houses
		{time: mustParseTime("2023-09-12T23:10:00+01:00"), bessSoe: 100, consumerDemand: 0, imbalancePrice: 60, siteImportPowerLimit: &fifty, siteExportPowerLimit: &seventy, expectedBessTargetPower: 70},
		{time: mustParseTime("2023-09-12T23:10:00+01:00"), bessSoe: 100, consumerDemand: -10, imbalancePrice: 60, siteImportPowerLimit: &fifty, siteExportPowerLimit: &seventy, expectedBessTargetPower: 60},
		{time: mustParseTime("2023-09-12T23:10:00+01:00"), bessSoe: 100, consumerDemand: -50, imbalancePrice: 60, siteImportPowerLimit: &fifty, siteExportPowerLimit: &seventy, expectedBessTargetPower: 20},
		{time: mustParseTime("2023-09-12T23:10:00+01:00"), bessSoe: 100, consumerDemand: 10, imbalancePrice: 60, siteImportPowerLimit: &fifty, siteExportPowerLimit: &seventy, expectedBessTargetPower: 80},
		{time: mustParseTime("2023-09-12T23:10:00+01:00"), bessSoe: 100, consumerDemand: 100, imbalancePrice: 60, siteImportPowerLimit: &fifty, siteExportPowerLimit: &seventy, expectedBessTargetPower: 105}, // here we are limited by the discharge power limits of the BESS

		// Imbalance price is attractive for charge - DUoS charges plus imbalance is 0p/kWh - but we are limited by site import limits, which also track any load/generation from the houses
		{time: mustParseTime("2023-09-12T23:10:00+01:00"), bessSoe: 100, consumerDemand: 0, imbalancePrice: -10, siteImportPowerLimit: &fifty, siteExportPowerLimit: &seventy, expectedBessTargetPower: -50},
		{time: mustParseTime("2023-09-12T23:10:00+01:00"), bessSoe: 100, consumerDemand: 10, imbalancePrice: -10, siteImportPowerLimit: &fifty, siteExportPowerLimit: &seventy, expectedBessTargetPower: -40},
		{time: mustParseTime("2023-09-12T23:10:00+01:00"), bessSoe: 100, consumerDemand: 45, imbalancePrice: -10, siteImportPowerLimit: &fifty, siteExportPowerLimit: &seventy, expectedBessTargetPower: -5},
		{time: mustParseTime("2023-09-12T23:10:00+01:00"), bessSoe: 100, consumerDemand: -10, imbalancePrice: -10, siteImportPowerLimit: &fifty, siteExportPowerLimit: &seventy, expectedBessTargetPower: -60},
		{time: mustParseTime("2023-09-12T23:10:00+01:00"), bessSoe: 100, consumerDemand: -30, imbalancePrice: -10, siteImportPowerLimit: &fifty, siteExportPowerLimit: &seventy, expectedBessTargetPower: -80},
		{time: mustParseTime("2023-09-12T23:10:00+01:00"), bessSoe: 100, consumerDemand: -60, imbalancePrice: -10, siteImportPowerLimit: &fifty, siteExportPowerLimit: &seventy, expectedBessTargetPower: -100}, // here we are limited by the charge power limits of the BESS
	}

	mock := microgridMock{
		SiteMeterReadings: ctrl.SiteMeterReadings,
		BessReadings:      ctrl.BessReadings,
		BessCommands:      bessCommands,
	}
	for _, point := range testpoints {
		test.Logf("Simulating time %v", point.time)

		// Update the mock modo client to return the test point's imbalance price
		ctrl.config.ModoClient = &MockImbalancePricer{
			price:  point.imbalancePrice,
			volume: point.imbalanceVolume,
			time:   timeutils.FloorHH(point.time),
		}

		if point.siteImportPowerLimit == nil {
			ctrl.config.SiteImportPowerLimit = 9999
		} else {
			ctrl.config.SiteImportPowerLimit = *point.siteImportPowerLimit
		}
		if point.siteExportPowerLimit == nil {
			ctrl.config.SiteExportPowerLimit = 9999
		} else {
			ctrl.config.SiteExportPowerLimit = *point.siteExportPowerLimit
		}

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

type MockImbalancePricer struct {
	price  float64
	volume float64
	time   time.Time
}

func (m *MockImbalancePricer) ImbalancePrice() (float64, time.Time) {
	return m.price, m.time
}

func (m *MockImbalancePricer) ImbalanceVolume() (float64, time.Time) {
	return m.volume, m.time
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
