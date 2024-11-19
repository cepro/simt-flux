package controller

import (
	"math"
	"testing"
	"time"

	"github.com/cepro/besscontroller/config"
	timeutils "github.com/cepro/besscontroller/time_utils"
)

func TestDynamicPeakDischarge(test *testing.T) {

	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		test.Fatalf("Could not load location: %v", err)
	}

	configs := []config.DynamicPeakDischargeConfig{
		{
			DayedPeriod: timeutils.DayedPeriod{
				Days: timeutils.Days{
					Name:     timeutils.AllDaysName,
					Location: london,
				},
				ClockTimePeriod: timeutils.ClockTimePeriod{
					Start: timeutils.ClockTime{Hour: 17, Minute: 0, Second: 0, Location: london},
					End:   timeutils.ClockTime{Hour: 19, Minute: 0, Second: 0, Location: london},
				},
			},
			TargetSoe:          100,
			TargetShortPeriods: true,
			ShortPrediction: config.NivPredictionDirectionConfig{
				AllowPrediction: true,
				VolumeCutoff:    0,
				TimeCutoffSecs:  1200, // 20 mins
			},
			PrioritiseResidualLoad: true,
		},
	}

	type subTest struct {
		name                     string
		t                        time.Time
		bessSoe                  float64
		sitePower                float64
		lastTargetPower          float64
		maxBessDischarge         float64
		imbalanceVolume          float64
		prioritiseResidualLoad   bool
		expectedControlComponent controlComponent
	}

	inactiveComponent := controlComponent{isActive: false}
	maxDischargeComponent := controlComponent{
		name:         "dynamic_peak_discharge",
		isActive:     true,
		targetPower:  math.Inf(1),
		controlPoint: controlPointBess,
	}
	importAvoidanceComponent := controlComponent{
		name:         "dynamic_peak_discharge",
		isActive:     true,
		targetPower:  0,
		controlPoint: controlPointSite,
	}

	subTests := []subTest{
		{
			name:                     "Outside of peak: nothing happens",
			t:                        mustParseTime("2024-09-05T13:51:00+01:00"),
			bessSoe:                  500.0,
			sitePower:                0,
			lastTargetPower:          0,
			maxBessDischarge:         100,
			imbalanceVolume:          0.0,
			prioritiseResidualLoad:   false,
			expectedControlComponent: inactiveComponent,
		},
		{
			name:                     "Surplus energy: discharge at max rate",
			t:                        mustParseTime("2024-09-05T17:10:00+01:00"),
			bessSoe:                  500.0,
			sitePower:                10, // site is only drawing 10kW, so no need to reserve any energy as we have 500kWh
			lastTargetPower:          0,
			maxBessDischarge:         100,
			imbalanceVolume:          0.0,
			prioritiseResidualLoad:   false,
			expectedControlComponent: maxDischargeComponent,
		},
		{
			name:                     "Scarce energy, long system, don't prioritise resid. load: dont dischage",
			t:                        mustParseTime("2024-09-05T17:10:00+01:00"),
			bessSoe:                  500.0,
			sitePower:                10,
			lastTargetPower:          0,
			maxBessDischarge:         400,
			imbalanceVolume:          -10,
			prioritiseResidualLoad:   false,
			expectedControlComponent: inactiveComponent,
		},
		{
			name:                     "Scarce energy, long system, prioritise resid. load: discharge to match load",
			t:                        mustParseTime("2024-09-05T17:10:00+01:00"),
			bessSoe:                  500.0,
			sitePower:                10,
			lastTargetPower:          0,
			maxBessDischarge:         400,
			imbalanceVolume:          -10,
			prioritiseResidualLoad:   true,
			expectedControlComponent: importAvoidanceComponent,
		},
		{
			name:                     "Scarce energy, short system, don't prioritise resid. load: discharge at max",
			t:                        mustParseTime("2024-09-05T17:10:00+01:00"),
			bessSoe:                  500.0,
			sitePower:                10,
			lastTargetPower:          0,
			maxBessDischarge:         400,
			imbalanceVolume:          50,
			prioritiseResidualLoad:   false,
			expectedControlComponent: maxDischargeComponent,
		},
		{
			name:                     "Scarce energy, but more than 'reserve for resid. load', short system, prioritise resid. load: discharge at max rate",
			t:                        mustParseTime("2024-09-05T17:10:00+01:00"),
			bessSoe:                  500.0,
			sitePower:                10,
			lastTargetPower:          0,
			maxBessDischarge:         400,
			imbalanceVolume:          50,
			prioritiseResidualLoad:   true,
			expectedControlComponent: maxDischargeComponent,
		},
		{
			name:                     "Scarce energy, less than 'reserve for resid. load', short system, prioritise resid. load: do import avoidance",
			t:                        mustParseTime("2024-09-05T17:10:00+01:00"),
			bessSoe:                  117,
			sitePower:                10,
			lastTargetPower:          0,
			maxBessDischarge:         400,
			imbalanceVolume:          50,
			prioritiseResidualLoad:   true,
			expectedControlComponent: importAvoidanceComponent,
		},
	}
	for _, subTest := range subTests {
		test.Run(subTest.name, func(t *testing.T) {

			// update the configs for this subtest
			for i := range configs {
				configs[i].PrioritiseResidualLoad = subTest.prioritiseResidualLoad
			}
			component := dynamicPeakDischarge(
				subTest.t,
				configs,
				subTest.bessSoe,
				subTest.sitePower,
				subTest.lastTargetPower,
				subTest.maxBessDischarge,
				&MockImbalancePricer{
					price:  0.0,
					volume: subTest.imbalanceVolume,
					time:   timeutils.FloorHH(subTest.t),
				},
			)

			if !componentsEquivalent(component, subTest.expectedControlComponent) {
				t.Errorf("got %v, expected %v", component, subTest.expectedControlComponent)
			}
		})
	}

}

func TestDynamicPeakApproach(test *testing.T) {

	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		test.Fatalf("Could not load location: %v", err)
	}

	configs := []config.DynamicPeakApproachConfig{
		{
			PeakPeriod: timeutils.DayedPeriod{
				Days: timeutils.Days{
					Name:     timeutils.AllDaysName,
					Location: london,
				},
				ClockTimePeriod: timeutils.ClockTimePeriod{
					Start: timeutils.ClockTime{Hour: 17, Minute: 0, Second: 0, Location: london},
					End:   timeutils.ClockTime{Hour: 19, Minute: 0, Second: 0, Location: london},
				},
			},
			ToSoe:                         1000,
			AssumedChargePower:            500, // Time to charge from 0 -> 1000kWh = 2hrs
			ForceChargeDurationFactor:     1.0, // Forcing starts at: 5pm - 30mins - 1 * 2hrs = 2:30pm
			EncourageChargeDurationFactor: 2.0, // Encouraging starts at: 5pm - 30mins - 2 * 2hrs = 12:30pm
			ChargeCushionMins:             30,
			LongPrediction: config.NivPredictionDirectionConfig{
				AllowPrediction: true,
				VolumeCutoff:    0,
				TimeCutoffSecs:  1200, // 20 mins
			},
		},
	}

	type subTest struct {
		name                     string
		t                        time.Time
		bessSoe                  float64
		imbalanceVolume          float64
		expectedControlComponent controlComponent
	}

	inactiveComponent := controlComponent{isActive: false}

	subTests := []subTest{
		{
			name:                     "Outside of peak approach: nothing happens",
			t:                        mustParseTime("2024-09-05T09:25:00+01:00"),
			bessSoe:                  0.0,
			imbalanceVolume:          -100,
			expectedControlComponent: inactiveComponent,
		},
		{
			name:                     "Within 'encourage zone' but short: nothing happens",
			t:                        mustParseTime("2024-09-05T13:40:00+01:00"),
			bessSoe:                  0.0,
			imbalanceVolume:          100,
			expectedControlComponent: inactiveComponent,
		},
		{
			name:                     "Within 'encourage zone' and long: charge 1",
			t:                        mustParseTime("2024-09-05T12:40:00+01:00"),
			bessSoe:                  0.0,
			imbalanceVolume:          -100,
			expectedControlComponent: controlComponent{name: "dynamic_peak_approach", isActive: true, targetPower: -125.0, controlPoint: controlPointBess},
		},
		{
			name:                     "Within 'encourage zone' and long: charge 2",
			t:                        mustParseTime("2024-09-05T13:40:00+01:00"),
			bessSoe:                  0.0,
			imbalanceVolume:          -100,
			expectedControlComponent: controlComponent{name: "dynamic_peak_approach", isActive: true, targetPower: -875.0, controlPoint: controlPointBess},
		},
		{
			name:                     "Within 'force zone' and short: charge to force curve",
			t:                        mustParseTime("2024-09-05T14:40:00+01:00"),
			bessSoe:                  0.0,
			imbalanceVolume:          100,
			expectedControlComponent: controlComponent{name: "dynamic_peak_approach", isActive: true, targetPower: -250.0, controlPoint: controlPointBess},
		},
		{
			name:                     "Within 'force zone' and short: charge to force curve 2",
			t:                        mustParseTime("2024-09-05T14:40:00+01:00"),
			bessSoe:                  10.0,
			imbalanceVolume:          100,
			expectedControlComponent: controlComponent{name: "dynamic_peak_approach", isActive: true, targetPower: -220.0, controlPoint: controlPointBess},
		},
		{
			name:                     "Within 'force zone' and long: charge to encourage curve",
			t:                        mustParseTime("2024-09-05T14:40:00+01:00"),
			bessSoe:                  0.0,
			imbalanceVolume:          -100,
			expectedControlComponent: controlComponent{name: "dynamic_peak_approach", isActive: true, targetPower: -1625.0, controlPoint: controlPointBess},
		},
		{
			name:                     "Within 'encourage zone' and short: charge",
			t:                        mustParseTime("2024-09-05T12:40:00+01:00"),
			bessSoe:                  0.0,
			imbalanceVolume:          -100,
			expectedControlComponent: controlComponent{name: "dynamic_peak_approach", isActive: true, targetPower: -125.0, controlPoint: controlPointBess},
		},
	}
	for _, subTest := range subTests {
		test.Run(subTest.name, func(t *testing.T) {

			component := dynamicPeakApproach(
				subTest.t,
				configs,
				subTest.bessSoe,
				1.0,
				&MockImbalancePricer{
					price:  0.0,
					volume: subTest.imbalanceVolume,
					time:   timeutils.FloorHH(subTest.t),
				},
			)

			if !componentsEquivalent(component, subTest.expectedControlComponent) {
				t.Errorf("got %v, expected %v", component, subTest.expectedControlComponent)
			}
		})
	}

}
