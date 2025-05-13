package controller

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cepro/besscontroller/axle"
	"github.com/cepro/besscontroller/cartesian"
	"github.com/cepro/besscontroller/config"
	"github.com/cepro/besscontroller/telemetry"
	timeutils "github.com/cepro/besscontroller/time_utils"
)

const (
	chargeEfficiency = 0.9
)

// baseTestInitialisation creates a basic configuration and the context and channels that are required for all tests.
func baseTestInitialisation() (Config, context.Context, chan telemetry.BessCommand, chan time.Time) {

	ctx := context.Background()
	bessCommandsChan := make(chan telemetry.BessCommand, 1)
	ctrlTickerChan := make(chan time.Time, 1)

	// Create a base controller configuration
	baseConfig := Config{
		BessChargeEfficiency:    chargeEfficiency,
		BessSoeMin:              20,
		BessSoeMax:              180,
		BessChargePowerLimit:    100,
		BessDischargePowerLimit: 105,  // slightly higher discharge limit than charge limit for testing
		SiteImportPowerLimit:    9999, // replaced at test time
		SiteExportPowerLimit:    9999, // replaced at test time
		ModoClient:              &MockImbalancePricer{},
		MaxReadingAge:           5 * time.Second,
		BessCommands:            bessCommandsChan,
	}

	return baseConfig, ctx, bessCommandsChan, ctrlTickerChan
}

// TestController is a high level (almost integration) test of the controller's ability
// to issue BessCommands to service various control modes.
func TestController(test *testing.T) {
	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		test.Fatalf("Could not load location: %v", err)
	}

	// pre-define some useful values
	weekdays := timeutils.Days{
		Name:     timeutils.WeekdayDaysName,
		Location: london,
	}
	alldays := timeutils.Days{
		Name:     timeutils.WeekdayDaysName,
		Location: london,
	}

	// Test import avoidance
	test.Run("ImportAvoidance", func(t *testing.T) {
		importAvoidancePeriods := []timeutils.DayedPeriod{
			{
				Days: weekdays,
				ClockTimePeriod: timeutils.ClockTimePeriod{
					Start: timeutils.ClockTime{Hour: 9, Minute: 0, Second: 0, Location: london},
					End:   timeutils.ClockTime{Hour: 10, Minute: 0, Second: 0, Location: london},
				},
			},
			// {  // TODO: remove
			// 	Days: weekdays,
			// 	ClockTimePeriod: timeutils.ClockTimePeriod{
			// 		Start: timeutils.ClockTime{Hour: 15, Minute: 0, Second: 0, Location: london},
			// 		End:   timeutils.ClockTime{Hour: 16, Minute: 0, Second: 0, Location: london},
			// 	},
			// },
			// {
			// 	Days: weekdays,
			// 	ClockTimePeriod: timeutils.ClockTimePeriod{
			// 		Start: timeutils.ClockTime{Hour: 21, Minute: 0, Second: 0, Location: london},
			// 		End:   timeutils.ClockTime{Hour: 22, Minute: 0, Second: 0, Location: london},
			// 	},
			// },
			// {
			// 	Days: weekdays,
			// 	ClockTimePeriod: timeutils.ClockTimePeriod{
			// 		Start: timeutils.ClockTime{Hour: 23, Minute: 30, Second: 0, Location: london},
			// 		End:   timeutils.ClockTime{Hour: 23, Minute: 59, Second: 59, Location: london},
			// 	},
			// },
		}

		config, ctx, bessCommandsChan, ctrlTickerChan := baseTestInitialisation()
		config.ImportAvoidancePeriods = importAvoidancePeriods

		ctrl := New(config)
		go ctrl.Run(ctx, ctrlTickerChan)
		mock := microgridMock{
			SiteMeterReadings: ctrl.SiteMeterReadings,
			BessReadings:      ctrl.BessReadings,
			BessCommands:      bessCommandsChan,
		}

		testPoints := []testpoint{

			// Outside of configured time - do nothing
			{time: mustParseTime("2023-09-12T08:45:00+01:00"), bessSoe: 150, consumerDemand: 25, expectedBessTargetPower: 0},

			// A period of increasing demand whilst we are in 'import avoidance' - the controller should use the battery to match the demand
			{time: mustParseTime("2023-09-12T09:00:04+01:00"), bessSoe: 150, consumerDemand: 25, expectedBessTargetPower: 25},
			{time: mustParseTime("2023-09-12T09:00:05+01:00"), bessSoe: 149, consumerDemand: 50, expectedBessTargetPower: 50},
			{time: mustParseTime("2023-09-12T09:00:06+01:00"), bessSoe: 147, consumerDemand: 75, expectedBessTargetPower: 75},
			{time: mustParseTime("2023-09-12T09:00:07+01:00"), bessSoe: 145, consumerDemand: 100, expectedBessTargetPower: 100},

			// Demand exceeds battery capability during 'import avoidance' - should stick to max battery power
			{time: mustParseTime("2023-09-12T09:00:08+01:00"), bessSoe: 143, consumerDemand: 110, expectedBessTargetPower: 105},
			{time: mustParseTime("2023-09-12T09:00:09+01:00"), bessSoe: 141, consumerDemand: 120, expectedBessTargetPower: 105},
			{time: mustParseTime("2023-09-12T09:00:10+01:00"), bessSoe: 139, consumerDemand: 106, expectedBessTargetPower: 105},

			// Decreasing demand during 'import avoidance' - controller backs off to match demand
			{time: mustParseTime("2023-09-12T09:00:11+01:00"), bessSoe: 144, consumerDemand: 50, expectedBessTargetPower: 50},
			{time: mustParseTime("2023-09-12T09:00:12+01:00"), bessSoe: 142, consumerDemand: 25, expectedBessTargetPower: 25},

			// Zero demand during 'import avoidance'
			{time: mustParseTime("2023-09-12T09:00:13+01:00"), bessSoe: 141, consumerDemand: 0, expectedBessTargetPower: 0},
			{time: mustParseTime("2023-09-12T09:00:14+01:00"), bessSoe: 141, consumerDemand: 0, expectedBessTargetPower: 0},
			{time: mustParseTime("2023-09-12T09:00:15+01:00"), bessSoe: 141, consumerDemand: 0, expectedBessTargetPower: 0},

			// Solar surplus during 'import avoidance' - should allow export
			{time: mustParseTime("2023-09-12T09:00:16+01:00"), bessSoe: 141, consumerDemand: -10, expectedBessTargetPower: 0},
			{time: mustParseTime("2023-09-12T09:00:17+01:00"), bessSoe: 142, consumerDemand: -10, expectedBessTargetPower: 0},

			// Weekend case - import avoidance should not apply
			{time: mustParseTime("2023-09-09T09:00:06+01:00"), bessSoe: 147, consumerDemand: 75, expectedBessTargetPower: 0},

			// Outside of configured time - do nothing
			{time: mustParseTime("2023-09-12T10:01:00+01:00"), bessSoe: 150, consumerDemand: 25, expectedBessTargetPower: 0},
		}

		runTestScenario(t, &mock, ctrlTickerChan, ctrl, testPoints)
	})

	// Test export avoidance, where the controller prevents grid exports
	test.Run("ExportAvoidance", func(t *testing.T) {
		exportAvoidancePeriods := []timeutils.DayedPeriod{
			{
				Days: alldays,
				ClockTimePeriod: timeutils.ClockTimePeriod{
					Start: timeutils.ClockTime{Hour: 11, Minute: 0, Second: 0, Location: london},
					End:   timeutils.ClockTime{Hour: 12, Minute: 0, Second: 0, Location: london},
				},
			},
			// {  // TODO: remove these
			// 	Days: alldays,
			// 	ClockTimePeriod: timeutils.ClockTimePeriod{
			// 		Start: timeutils.ClockTime{Hour: 15, Minute: 0, Second: 0, Location: london},
			// 		End:   timeutils.ClockTime{Hour: 16, Minute: 0, Second: 0, Location: london},
			// 	},
			// },
			// {
			// 	Days: alldays,
			// 	ClockTimePeriod: timeutils.ClockTimePeriod{
			// 		Start: timeutils.ClockTime{Hour: 17, Minute: 0, Second: 0, Location: london},
			// 		End:   timeutils.ClockTime{Hour: 18, Minute: 0, Second: 0, Location: london},
			// 	},
			// },
			// {
			// 	Days: alldays,
			// 	ClockTimePeriod: timeutils.ClockTimePeriod{
			// 		Start: timeutils.ClockTime{Hour: 21, Minute: 0, Second: 0, Location: london},
			// 		End:   timeutils.ClockTime{Hour: 22, Minute: 0, Second: 0, Location: london},
			// 	},
			// },
		}

		config, ctx, bessCommandsChan, ctrlTickerChan := baseTestInitialisation()
		config.ExportAvoidancePeriods = exportAvoidancePeriods

		ctrl := New(config)
		go ctrl.Run(ctx, ctrlTickerChan)
		mock := microgridMock{
			SiteMeterReadings: ctrl.SiteMeterReadings,
			BessReadings:      ctrl.BessReadings,
			BessCommands:      bessCommandsChan,
		}

		testPoints := []testpoint{
			// Outside of configured time - do nothing
			{time: mustParseTime("2023-09-12T10:40:00+01:00"), bessSoe: 100, consumerDemand: -10, expectedBessTargetPower: 0},

			// Do export avoidance
			{time: mustParseTime("2023-09-12T11:00:00+01:00"), bessSoe: 100, consumerDemand: 15, expectedBessTargetPower: 0},
			{time: mustParseTime("2023-09-12T11:00:01+01:00"), bessSoe: 100, consumerDemand: 0, expectedBessTargetPower: 0},
			{time: mustParseTime("2023-09-12T11:00:02+01:00"), bessSoe: 100, consumerDemand: -10, expectedBessTargetPower: -10},
			{time: mustParseTime("2023-09-12T11:00:03+01:00"), bessSoe: 101, consumerDemand: -50, expectedBessTargetPower: -50},
			{time: mustParseTime("2023-09-12T11:00:04+01:00"), bessSoe: 102, consumerDemand: -500, expectedBessTargetPower: -100},
			{time: mustParseTime("2023-09-12T11:00:05+01:00"), bessSoe: 103, consumerDemand: 15, expectedBessTargetPower: 0},

			// Outside of configured time - do nothing
			{time: mustParseTime("2023-09-12T12:15:00+01:00"), bessSoe: 100, consumerDemand: -10, expectedBessTargetPower: 0},
		}

		runTestScenario(t, &mock, ctrlTickerChan, ctrl, testPoints)
	})

	// Test charge to battery SoE where the controller charges to reach some target SoE
	test.Run("ChargeToSoE", func(t *testing.T) {
		chargeToSoePeriods := []config.DayedPeriodWithSoe{
			{
				Soe: 130,
				DayedPeriod: timeutils.DayedPeriod{
					Days: alldays,
					ClockTimePeriod: timeutils.ClockTimePeriod{
						Start: timeutils.ClockTime{Hour: 13, Minute: 0, Second: 0, Location: london},
						End:   timeutils.ClockTime{Hour: 13, Minute: 30, Second: 0, Location: london},
					},
				},
			},
		}

		config, ctx, bessCommandsChan, ctrlTickerChan := baseTestInitialisation()
		config.ChargeToSoePeriods = chargeToSoePeriods

		ctrl := New(config)
		go ctrl.Run(ctx, ctrlTickerChan)
		mock := microgridMock{
			SiteMeterReadings: ctrl.SiteMeterReadings,
			BessReadings:      ctrl.BessReadings,
			BessCommands:      bessCommandsChan,
		}

		testPoints := []testpoint{
			// Outside of configured time - do nothing
			{time: mustParseTime("2023-09-12T12:00:00+01:00"), bessSoe: 100, consumerDemand: 15, expectedBessTargetPower: 0},

			// We need to make up 30kWh in the next 30mins
			{time: mustParseTime("2023-09-12T13:00:00+01:00"), bessSoe: 100, consumerDemand: 15, expectedBessTargetPower: -60 / chargeEfficiency},
			{time: mustParseTime("2023-09-12T13:00:01+01:00"), bessSoe: 100, consumerDemand: 0, expectedBessTargetPower: -60 / chargeEfficiency},
			{time: mustParseTime("2023-09-12T13:00:02+01:00"), bessSoe: 100, consumerDemand: -15, expectedBessTargetPower: -60 / chargeEfficiency},

			// We need to make up 15kWh in the next 30mins
			{time: mustParseTime("2023-09-12T13:00:03+01:00"), bessSoe: 115, consumerDemand: -15, expectedBessTargetPower: -30 / chargeEfficiency},

			// Outside of configured time - do nothing
			{time: mustParseTime("2023-09-12T13:40:00+01:00"), bessSoe: 100, consumerDemand: 15, expectedBessTargetPower: 0},
		}

		runTestScenario(t, &mock, ctrlTickerChan, ctrl, testPoints)
	})

	// Test discharge to battery target
	test.Run("DischargeToSoE", func(t *testing.T) {
		// Configure discharge to SoE periods
		dischargeToSoePeriods := []config.DayedPeriodWithSoe{
			{
				Soe: 70,
				DayedPeriod: timeutils.DayedPeriod{
					Days: alldays,
					ClockTimePeriod: timeutils.ClockTimePeriod{
						Start: timeutils.ClockTime{Hour: 13, Minute: 30, Second: 0, Location: london},
						End:   timeutils.ClockTime{Hour: 14, Minute: 0, Second: 0, Location: london},
					},
				},
			},
		}

		config, ctx, bessCommandsChan, ctrlTickerChan := baseTestInitialisation()
		config.DischargeToSoePeriods = dischargeToSoePeriods

		ctrl := New(config)
		go ctrl.Run(ctx, ctrlTickerChan)
		mock := microgridMock{
			SiteMeterReadings: ctrl.SiteMeterReadings,
			BessReadings:      ctrl.BessReadings,
			BessCommands:      bessCommandsChan,
		}

		testPoints := []testpoint{

			// Weekend case - discharge to SoE should not apply
			{time: mustParseTime("2023-09-10T13:30:02+01:00"), bessSoe: 100, consumerDemand: 15, expectedBessTargetPower: 0},

			// Outside of configured time - do nothing
			{time: mustParseTime("2023-09-12T13:00:00+01:00"), bessSoe: 100, consumerDemand: 15, expectedBessTargetPower: 0},

			// Discharge to SoE - controller discharges to reach target SoE
			{time: mustParseTime("2023-09-12T13:30:00+01:00"), bessSoe: 100, consumerDemand: 15, expectedBessTargetPower: 60},
			{time: mustParseTime("2023-09-12T13:30:01+01:00"), bessSoe: 200, consumerDemand: 0, expectedBessTargetPower: 105},
			{time: mustParseTime("2023-09-12T13:30:02+01:00"), bessSoe: 100, consumerDemand: 15, expectedBessTargetPower: 60},

			// Outside of configured time - do nothing
			{time: mustParseTime("2023-09-12T14:05:00+01:00"), bessSoe: 100, consumerDemand: 15, expectedBessTargetPower: 0},
		}

		runTestScenario(t, &mock, ctrlTickerChan, ctrl, testPoints)
	})

	// Test multiple active modes
	test.Run("MultipleModes", func(t *testing.T) {
		// Configure periods for both import and export avoidance
		importAvoidancePeriods := []timeutils.DayedPeriod{
			{
				Days: weekdays,
				ClockTimePeriod: timeutils.ClockTimePeriod{
					Start: timeutils.ClockTime{Hour: 15, Minute: 0, Second: 0, Location: london},
					End:   timeutils.ClockTime{Hour: 16, Minute: 0, Second: 0, Location: london},
				},
			},
		}

		exportAvoidancePeriods := []timeutils.DayedPeriod{
			{
				Days: alldays,
				ClockTimePeriod: timeutils.ClockTimePeriod{
					Start: timeutils.ClockTime{Hour: 15, Minute: 0, Second: 0, Location: london},
					End:   timeutils.ClockTime{Hour: 16, Minute: 0, Second: 0, Location: london},
				},
			},
			{
				Days: alldays,
				ClockTimePeriod: timeutils.ClockTimePeriod{
					Start: timeutils.ClockTime{Hour: 17, Minute: 0, Second: 0, Location: london},
					End:   timeutils.ClockTime{Hour: 18, Minute: 0, Second: 0, Location: london},
				},
			},
		}

		chargeToSoePeriods := []config.DayedPeriodWithSoe{
			{
				Soe: 190,
				DayedPeriod: timeutils.DayedPeriod{
					Days: alldays,
					ClockTimePeriod: timeutils.ClockTimePeriod{
						Start: timeutils.ClockTime{Hour: 17, Minute: 0, Second: 0, Location: london},
						End:   timeutils.ClockTime{Hour: 18, Minute: 0, Second: 0, Location: london},
					},
				},
			},
		}

		config, ctx, bessCommandsChan, ctrlTickerChan := baseTestInitialisation()
		config.ImportAvoidancePeriods = importAvoidancePeriods
		config.ExportAvoidancePeriods = exportAvoidancePeriods
		config.ChargeToSoePeriods = chargeToSoePeriods

		ctrl := New(config)
		go ctrl.Run(ctx, ctrlTickerChan)
		mock := microgridMock{
			SiteMeterReadings: ctrl.SiteMeterReadings,
			BessReadings:      ctrl.BessReadings,
			BessCommands:      bessCommandsChan,
		}

		testPoints := []testpoint{
			// Outside of configured times - do nothing
			{time: mustParseTime("2023-09-12T14:00:00+01:00"), bessSoe: 160, consumerDemand: 15, expectedBessTargetPower: 0},

			// Test when both 'export avoidance' and 'import avoidance' are active
			{time: mustParseTime("2023-09-12T15:00:00+01:00"), bessSoe: 160, consumerDemand: 15, expectedBessTargetPower: 15},
			{time: mustParseTime("2023-09-12T15:00:01+01:00"), bessSoe: 160, consumerDemand: 0, expectedBessTargetPower: 0},
			{time: mustParseTime("2023-09-12T15:00:02+01:00"), bessSoe: 160, consumerDemand: -15, expectedBessTargetPower: -15},

			// Test when both 'export avoidance' and 'charge to min' are active -  the controller should use the 'charge to min' value as a minimum but allow 'export avoidance' to specify a larger charge
			{time: mustParseTime("2023-09-12T17:00:00+01:00"), bessSoe: 160, consumerDemand: 15, expectedBessTargetPower: -30 / chargeEfficiency},
			{time: mustParseTime("2023-09-12T17:00:01+01:00"), bessSoe: 160, consumerDemand: 0, expectedBessTargetPower: -30 / chargeEfficiency},
			{time: mustParseTime("2023-09-12T17:00:02+01:00"), bessSoe: 160, consumerDemand: -15, expectedBessTargetPower: -30 / chargeEfficiency},
			{time: mustParseTime("2023-09-12T17:00:03+01:00"), bessSoe: 160, consumerDemand: -100, expectedBessTargetPower: -100},

			// Outside of configured times - do nothing
			{time: mustParseTime("2023-09-12T21:00:00+01:00"), bessSoe: 160, consumerDemand: 15, expectedBessTargetPower: 0},
		}

		runTestScenario(t, &mock, ctrlTickerChan, ctrl, testPoints)
	})

	// Test that the SoE limits are respected
	test.Run("SoELimits", func(t *testing.T) {
		// Configure both import and export avoidance periods for the test time
		importAvoidancePeriods := []timeutils.DayedPeriod{
			{
				Days: weekdays,
				ClockTimePeriod: timeutils.ClockTimePeriod{
					Start: timeutils.ClockTime{Hour: 21, Minute: 0, Second: 0, Location: london},
					End:   timeutils.ClockTime{Hour: 22, Minute: 0, Second: 0, Location: london},
				},
			},
		}
		exportAvoidancePeriods := []timeutils.DayedPeriod{
			{
				Days: alldays,
				ClockTimePeriod: timeutils.ClockTimePeriod{
					Start: timeutils.ClockTime{Hour: 21, Minute: 0, Second: 0, Location: london},
					End:   timeutils.ClockTime{Hour: 22, Minute: 0, Second: 0, Location: london},
				},
			},
		}

		config, ctx, bessCommandsChan, ctrlTickerChan := baseTestInitialisation()
		config.ImportAvoidancePeriods = importAvoidancePeriods
		config.ExportAvoidancePeriods = exportAvoidancePeriods

		ctrl := New(config)
		go ctrl.Run(ctx, ctrlTickerChan)
		mock := microgridMock{
			SiteMeterReadings: ctrl.SiteMeterReadings,
			BessReadings:      ctrl.BessReadings,
			BessCommands:      bessCommandsChan,
		}

		testPoints := []testpoint{
			// Ensure maximum bess SoE is honored (import and export avoidance active)
			{time: mustParseTime("2023-09-12T21:00:00+01:00"), bessSoe: 179, consumerDemand: -50, expectedBessTargetPower: -50}, // can charge 1kWh
			{time: mustParseTime("2023-09-12T21:00:01+01:00"), bessSoe: 180, consumerDemand: -50, expectedBessTargetPower: 0},   // can't charge anything as battery is full
			{time: mustParseTime("2023-09-12T21:00:02+01:00"), bessSoe: 180, consumerDemand: -50, expectedBessTargetPower: 0},
			{time: mustParseTime("2023-09-12T21:00:03+01:00"), bessSoe: 181, consumerDemand: -50, expectedBessTargetPower: 0},

			// Ensure minimum bess SoE is honored (import and export avoidance active)
			{time: mustParseTime("2023-09-12T21:30:00+01:00"), bessSoe: 21, consumerDemand: 10, expectedBessTargetPower: 10}, // can discharge 1kWh
			{time: mustParseTime("2023-09-12T21:30:01+01:00"), bessSoe: 20, consumerDemand: 10, expectedBessTargetPower: 0},  // can't discharge as battery is at the configured min SoE
			{time: mustParseTime("2023-09-12T21:30:02+01:00"), bessSoe: 20, consumerDemand: 10, expectedBessTargetPower: 0},
			{time: mustParseTime("2023-09-12T21:30:03+01:00"), bessSoe: 19, consumerDemand: 10, expectedBessTargetPower: 0},
		}

		runTestScenario(t, &mock, ctrlTickerChan, ctrl, testPoints)
	})

	// Test NIV chasing (there are other dedicated niv chase tests in `comp_niv_chase_test.go`)
	test.Run("NivChasing", func(t *testing.T) {
		// Configure NIV chase periods
		nivChasePeriods := []config.DayedPeriodWithNIV{
			{
				DayedPeriod: timeutils.DayedPeriod{
					Days: alldays,
					ClockTimePeriod: timeutils.ClockTimePeriod{
						Start: timeutils.ClockTime{Hour: 23, Minute: 0, Second: 0, Location: london},
						End:   timeutils.ClockTime{Hour: 23, Minute: 59, Second: 59, Location: london},
					},
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
					DefaultPricing:  []config.TimedRate{},
				},
			},
		}

		// Define rate periods for import/export
		chargesPeriods := []timeutils.DayedPeriod{nivChasePeriods[0].DayedPeriod}
		ratesImport := []config.TimedRate{
			{
				Rate:    10,
				Periods: chargesPeriods,
			},
		}
		ratesExport := []config.TimedRate{
			{
				Rate:    -10,
				Periods: chargesPeriods,
			},
		}

		config, ctx, bessCommandsChan, ctrlTickerChan := baseTestInitialisation()
		config.NivChasePeriods = nivChasePeriods
		config.RatesImport = ratesImport
		config.RatesExport = ratesExport

		ctrl := New(config)
		go ctrl.Run(ctx, ctrlTickerChan)
		mock := microgridMock{
			SiteMeterReadings: ctrl.SiteMeterReadings,
			BessReadings:      ctrl.BessReadings,
			BessCommands:      bessCommandsChan,
		}

		// These are defined as vars because Go cannot get the address of a constant - e.g. you can't do `&50.0`
		var fifty float64 = 50.0
		var seventy float64 = 70.0

		testPoints := []testpoint{
			// Imbalance price attractive for charge - charge at full rate with charge limits
			{time: mustParseTime("2023-09-12T23:10:00+01:00"), bessSoe: 0, consumerDemand: 10, imbalancePrice: -10, expectedBessTargetPower: -100},

			// Imbalance price attractive for discharge - discharge at full rate with discharge limits
			{time: mustParseTime("2023-09-12T23:10:00+01:00"), bessSoe: 180, consumerDemand: 10, imbalancePrice: 60, expectedBessTargetPower: 105},

			// Imbalance price attractive for discharge - limited by site export
			{time: mustParseTime("2023-09-12T23:10:00+01:00"), bessSoe: 100, consumerDemand: 0, imbalancePrice: 60, siteImportPowerLimit: &fifty, siteExportPowerLimit: &seventy, expectedBessTargetPower: 70},
			{time: mustParseTime("2023-09-12T23:10:00+01:00"), bessSoe: 100, consumerDemand: -10, imbalancePrice: 60, siteImportPowerLimit: &fifty, siteExportPowerLimit: &seventy, expectedBessTargetPower: 60},
			{time: mustParseTime("2023-09-12T23:10:00+01:00"), bessSoe: 100, consumerDemand: -50, imbalancePrice: 60, siteImportPowerLimit: &fifty, siteExportPowerLimit: &seventy, expectedBessTargetPower: 20},
			{time: mustParseTime("2023-09-12T23:10:00+01:00"), bessSoe: 100, consumerDemand: 10, imbalancePrice: 60, siteImportPowerLimit: &fifty, siteExportPowerLimit: &seventy, expectedBessTargetPower: 80},
			{time: mustParseTime("2023-09-12T23:10:00+01:00"), bessSoe: 100, consumerDemand: 100, imbalancePrice: 60, siteImportPowerLimit: &fifty, siteExportPowerLimit: &seventy, expectedBessTargetPower: 105}, // limited by BESS discharge power

			// Imbalance price attractive for charge - limited by site import
			{time: mustParseTime("2023-09-12T23:10:00+01:00"), bessSoe: 100, consumerDemand: 0, imbalancePrice: -10, siteImportPowerLimit: &fifty, siteExportPowerLimit: &seventy, expectedBessTargetPower: -50},
			{time: mustParseTime("2023-09-12T23:10:00+01:00"), bessSoe: 100, consumerDemand: 10, imbalancePrice: -10, siteImportPowerLimit: &fifty, siteExportPowerLimit: &seventy, expectedBessTargetPower: -40},
			{time: mustParseTime("2023-09-12T23:10:00+01:00"), bessSoe: 100, consumerDemand: 45, imbalancePrice: -10, siteImportPowerLimit: &fifty, siteExportPowerLimit: &seventy, expectedBessTargetPower: -5},
			{time: mustParseTime("2023-09-12T23:10:00+01:00"), bessSoe: 100, consumerDemand: -10, imbalancePrice: -10, siteImportPowerLimit: &fifty, siteExportPowerLimit: &seventy, expectedBessTargetPower: -60},
			{time: mustParseTime("2023-09-12T23:10:00+01:00"), bessSoe: 100, consumerDemand: -30, imbalancePrice: -10, siteImportPowerLimit: &fifty, siteExportPowerLimit: &seventy, expectedBessTargetPower: -80},
			{time: mustParseTime("2023-09-12T23:10:00+01:00"), bessSoe: 100, consumerDemand: -60, imbalancePrice: -10, siteImportPowerLimit: &fifty, siteExportPowerLimit: &seventy, expectedBessTargetPower: -100}, // limited by BESS charge power
		}

		runTestScenario(t, &mock, ctrlTickerChan, ctrl, testPoints)
	})

	// Test Axle Schedule commands
	test.Run("AxleScheduleCommands", func(t *testing.T) {
		config, ctx, bessCommandsChan, ctrlTickerChan := baseTestInitialisation()

		ctrl := New(config)
		go ctrl.Run(ctx, ctrlTickerChan)
		mock := microgridMock{
			SiteMeterReadings: ctrl.SiteMeterReadings,
			BessReadings:      ctrl.BessReadings,
			BessCommands:      bessCommandsChan,
		}

		// Create axle schedule
		axleSchedule := axle.Schedule{
			ReceivedTime: time.Time{},
			Items: []axle.ScheduleItem{
				{
					Start:          mustParseTime("2023-09-13T09:00:00+01:00"),
					End:            mustParseTime("2023-09-13T09:05:00+01:00"),
					Action:         "charge_max",
					AllowDeviation: false,
				},
				{
					Start:          mustParseTime("2023-09-13T11:00:00+01:00"),
					End:            mustParseTime("2023-09-13T11:05:00+01:00"),
					Action:         "avoid_import",
					AllowDeviation: false,
				},
				{
					Start:          mustParseTime("2023-09-13T12:00:00+01:00"),
					End:            mustParseTime("2023-09-13T12:05:00+01:00"),
					Action:         "avoid_export",
					AllowDeviation: false,
				},
			},
		}

		ctrl.AxleSchedules <- axleSchedule

		// These are defined as vars because Go cannot get the address of a constant - e.g. you can't do `&50.0`
		var fifty float64 = 50.0
		var seventy float64 = 70.0

		testPoints := []testpoint{
			// Outside of configured times - do nothing
			{time: mustParseTime("2023-09-13T08:59:00+01:00"), bessSoe: 50, consumerDemand: 0, imbalancePrice: 60, siteImportPowerLimit: &fifty, siteExportPowerLimit: &seventy, expectedBessTargetPower: 0},

			// Test the charge_max command
			{time: mustParseTime("2023-09-13T09:00:00+01:00"), bessSoe: 50, consumerDemand: 0, imbalancePrice: 60, siteImportPowerLimit: &fifty, siteExportPowerLimit: &seventy, expectedBessTargetPower: -50}, // limit is from the site import constraint
			{time: mustParseTime("2023-09-13T09:01:00+01:00"), bessSoe: 55, consumerDemand: 0, imbalancePrice: 60, siteImportPowerLimit: nil, siteExportPowerLimit: &seventy, expectedBessTargetPower: -100},   // limit is from the BESS charge power limit
			{time: mustParseTime("2023-09-13T09:02:00+01:00"), bessSoe: 180, consumerDemand: 0, imbalancePrice: 60, siteImportPowerLimit: nil, siteExportPowerLimit: &seventy, expectedBessTargetPower: 0},     // limit is from the BESS SoE

			// Test the avoid_import command
			{time: mustParseTime("2023-09-13T11:00:00+01:00"), bessSoe: 50, consumerDemand: -10, imbalancePrice: 60, siteImportPowerLimit: &fifty, siteExportPowerLimit: &seventy, expectedBessTargetPower: 0}, // nothing to do here as we are exporting
			{time: mustParseTime("2023-09-13T11:01:00+01:00"), bessSoe: 50, consumerDemand: 0, imbalancePrice: 60, siteImportPowerLimit: &fifty, siteExportPowerLimit: &seventy, expectedBessTargetPower: 0},   // nothing to do here as there is zero load / generation
			{time: mustParseTime("2023-09-13T11:02:00+01:00"), bessSoe: 50, consumerDemand: 10, imbalancePrice: 60, siteImportPowerLimit: &fifty, siteExportPowerLimit: &seventy, expectedBessTargetPower: 10}, // discharge to match the consumer demand - to avoid imports from the grid
			{time: mustParseTime("2023-09-13T11:03:00+01:00"), bessSoe: 0, consumerDemand: 10, imbalancePrice: 60, siteImportPowerLimit: &fifty, siteExportPowerLimit: &seventy, expectedBessTargetPower: 0},   // we would like to discharge to match the consumer demand, but there is no SoE left

			// Test the avoid_export command
			{time: mustParseTime("2023-09-13T12:01:00+01:00"), bessSoe: 50, consumerDemand: 0, imbalancePrice: 60, siteImportPowerLimit: &fifty, siteExportPowerLimit: &seventy, expectedBessTargetPower: 0},    // nothing to do here as there is zero load / generation
			{time: mustParseTime("2023-09-13T12:02:00+01:00"), bessSoe: 50, consumerDemand: 10, imbalancePrice: 60, siteImportPowerLimit: &fifty, siteExportPowerLimit: &seventy, expectedBessTargetPower: 10},  // discharge to match the consumer demand - to avoid imports from the grid
			{time: mustParseTime("2023-09-13T12:03:00+01:00"), bessSoe: 0, consumerDemand: -10, imbalancePrice: 60, siteImportPowerLimit: &fifty, siteExportPowerLimit: &seventy, expectedBessTargetPower: -10}, // charge to avoid export

			// Outside of configured times - do nothing
			{time: mustParseTime("2023-09-13T12:07:00+01:00"), bessSoe: 0, consumerDemand: -10, imbalancePrice: 60, siteImportPowerLimit: &fifty, siteExportPowerLimit: &seventy, expectedBessTargetPower: 0},
		}

		runTestScenario(t, &mock, ctrlTickerChan, ctrl, testPoints)
	})
}

// testpoint represents a point in time that we are testing as part of a larger timeseries
type testpoint struct {
	time                    time.Time // the point in time being tested
	bessSoe                 float64   // the state of energy of the battery at this point in time
	consumerDemand          float64   // the consumer demand at this point in time
	imbalancePrice          float64   // the predicted system settlement price for this period
	imbalanceVolume         float64   // the predicted system settlement volume for this period
	siteImportPowerLimit    *float64  // site import limits applied for the test point, defaults to 9999 if nil
	siteExportPowerLimit    *float64  // site export limits applied for the test point, defaults to 9999 if nil
	expectedBessTargetPower float64   // the power command that we expect the controller to issue at this point in time
}

// runTestScenario runs a batch of test points against the controller
func runTestScenario(t *testing.T, mock *microgridMock, ctrlTickerChan chan<- time.Time, ctrl *Controller, points []testpoint) {
	for _, point := range points {
		t.Logf("Simulating time %v", point.time)

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
			t.Errorf("At time '%v', failed to wait for bess command: %v", point.time, err)
			return
		}

		if !almostEqual(mock.bessTargetPower, point.expectedBessTargetPower, 0.1) {
			t.Errorf("At time '%v' got unexpected bess target power: %f, expected: %f", point.time, mock.bessTargetPower, point.expectedBessTargetPower)
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
