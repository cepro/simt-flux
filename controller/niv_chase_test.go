package controller

import (
	"testing"
	"time"

	"github.com/cepro/besscontroller/cartesian"
	"github.com/cepro/besscontroller/config"
	timeutils "github.com/cepro/besscontroller/time_utils"
)

func TestNivChase(test *testing.T) {

	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		test.Fatalf("Could not load location: %v", err)
	}

	chargeCurve1 := cartesian.Curve{
		Points: []cartesian.Point{
			{X: -9999, Y: 180},
			{X: 0, Y: 180},
			{X: 20, Y: 0},
		},
	}
	dischargeCurve1 := cartesian.Curve{
		Points: []cartesian.Point{
			{X: 30, Y: 180},
			{X: 40, Y: 0},
			{X: 9999, Y: 0},
		},
	}
	dischargeCurveWaterlilies := cartesian.Curve{
		Points: []cartesian.Point{
			{X: 40, Y: 444},
			{X: 40, Y: 0},
			{X: 999999999, Y: 0},
		},
	}

	nivChasePeriods := []config.DayedPeriodWithNIV{
		{
			DayedPeriod: timeutils.DayedPeriod{
				Days: timeutils.Days{
					Name:     timeutils.AllDaysName,
					Location: london,
				},
				ClockTimePeriod: timeutils.ClockTimePeriod{
					Start: timeutils.ClockTime{Hour: 23, Minute: 0, Second: 0, Location: london},
					End:   timeutils.ClockTime{Hour: 23, Minute: 59, Second: 59, Location: london},
				},
			},
			Niv: config.NivConfig{
				ChargeCurve:     cartesian.Curve{}, // adjusted dynamically in test
				DischargeCurve:  cartesian.Curve{}, // adjusted dynamically in test
				CurveShiftLong:  0,                 // adjusted dynamically in test
				CurveShiftShort: 0,                 // adjusted dynamically in test
				DefaultPricing:  []config.TimedRate{},
			},
		},
	}

	type subTest struct {
		name                     string
		t                        time.Time
		soe                      float64
		chargeCurve              cartesian.Curve
		dischargeCurve           cartesian.Curve
		curveShiftLong           float64
		curveShiftShort          float64
		imbalancePrice           float64
		imbalanceVolume          float64
		ratesImport              float64
		ratesExport              float64
		expectedControlComponent controlComponent
	}

	subTests := []subTest{
		{
			name:                     "Imbalance price is between the charge and discharge curves - no action",
			t:                        mustParseTime("2023-09-12T23:10:00+01:00"),
			soe:                      100.0,
			chargeCurve:              chargeCurve1,
			dischargeCurve:           dischargeCurve1,
			curveShiftLong:           0.0,
			curveShiftShort:          0.0,
			imbalancePrice:           25.0,
			imbalanceVolume:          0.0,
			expectedControlComponent: controlComponent{},
		},
		{
			name:                     "Imbalance price is attractive for charge - charge rate is set by curve following",
			t:                        mustParseTime("2023-09-12T23:10:00+01:00"),
			soe:                      160.0,
			chargeCurve:              chargeCurve1,
			dischargeCurve:           dischargeCurve1,
			curveShiftLong:           0.0,
			curveShiftShort:          0.0,
			imbalancePrice:           0.0,
			imbalanceVolume:          0.0,
			expectedControlComponent: activeControlComponent(-70.59),
		},
		{
			name:                     "Imbalance price is attractive for discharge - discharge rate is set by curve following",
			t:                        mustParseTime("2023-09-12T23:10:00+01:00"),
			soe:                      100.0,
			chargeCurve:              chargeCurve1,
			dischargeCurve:           dischargeCurve1,
			curveShiftLong:           0.0,
			curveShiftShort:          0.0,
			imbalancePrice:           35.0,
			imbalanceVolume:          0.0,
			expectedControlComponent: activeControlComponent(30.0),
		},
		{
			name:                     "Imbalance price is between the charge and discharge curves - but long rate shift triggers charge",
			t:                        mustParseTime("2023-09-12T23:10:00+01:00"),
			soe:                      160.0,
			chargeCurve:              chargeCurve1,
			dischargeCurve:           dischargeCurve1,
			curveShiftLong:           25.0,
			curveShiftShort:          25.0,
			imbalancePrice:           25.0,
			imbalanceVolume:          -100,
			expectedControlComponent: activeControlComponent(-70.59),
		},
		{
			name:                     "Imbalance price is between the charge and discharge curves - but short rate shift triggers discharge",
			t:                        mustParseTime("2023-09-12T23:10:00+01:00"),
			soe:                      100.0,
			chargeCurve:              chargeCurve1,
			dischargeCurve:           dischargeCurve1,
			curveShiftLong:           25.0,
			curveShiftShort:          25.0,
			imbalancePrice:           25.0,
			imbalanceVolume:          +100,
			expectedControlComponent: activeControlComponent(+300.0),
		},
		{
			name:                     "Imbalance price is attractive for charge - charge rate is set by curve following",
			t:                        mustParseTime("2023-09-12T23:10:00+01:00"),
			soe:                      160.0,
			chargeCurve:              chargeCurve1,
			dischargeCurve:           dischargeCurve1,
			curveShiftLong:           0.0,
			curveShiftShort:          0.0,
			imbalancePrice:           0.0,
			imbalanceVolume:          0.0,
			expectedControlComponent: activeControlComponent(-70.59),
		},
		{
			name:                     "Imbalance price is attractive for discharge - discharge rate is set by curve following",
			t:                        mustParseTime("2023-09-12T23:10:00+01:00"),
			soe:                      100.0,
			chargeCurve:              chargeCurve1,
			dischargeCurve:           dischargeCurve1,
			curveShiftLong:           0.0,
			curveShiftShort:          0.0,
			imbalancePrice:           35.0,
			imbalanceVolume:          0.0,
			expectedControlComponent: activeControlComponent(30.0),
		},
		{
			name:                     "Imbalance price is attractive for discharge - discharge rate is set by curve following, excentuated by short rate shift",
			t:                        mustParseTime("2023-09-12T23:10:00+01:00"),
			soe:                      100.0,
			chargeCurve:              chargeCurve1,
			dischargeCurve:           dischargeCurve1,
			curveShiftLong:           5,
			curveShiftShort:          5,
			imbalancePrice:           35.0,
			imbalanceVolume:          50,
			expectedControlComponent: activeControlComponent(300.0),
		},
		{
			name:                     "Imbalance price is between the charge and discharge curves - but DUoS charges fees trigger export",
			t:                        mustParseTime("2023-09-12T23:10:00+01:00"),
			soe:                      200.0,
			chargeCurve:              chargeCurve1,
			dischargeCurve:           dischargeCurve1,
			curveShiftLong:           0.0,
			curveShiftShort:          0.0,
			imbalancePrice:           25.0,
			imbalanceVolume:          0.0,
			ratesImport:              5,
			ratesExport:              -5,
			expectedControlComponent: activeControlComponent(60),
		},
		{
			name:                     "Test Waterlilies discharge curve - don't discharge as prices are moderatley high",
			t:                        mustParseTime("2023-09-12T23:10:00+01:00"),
			soe:                      200.0,
			chargeCurve:              cartesian.Curve{},
			dischargeCurve:           dischargeCurveWaterlilies,
			curveShiftLong:           6.0,
			curveShiftShort:          0.0,
			imbalancePrice:           25.0,
			imbalanceVolume:          500.0,
			ratesImport:              10,
			ratesExport:              -10,
			expectedControlComponent: controlComponent{},
		},
		{
			name:                     "Test Waterlilies discharge curve - discharge when prices are very high",
			t:                        mustParseTime("2023-09-12T23:10:00+01:00"),
			soe:                      200.0,
			chargeCurve:              cartesian.Curve{},
			dischargeCurve:           dischargeCurveWaterlilies,
			curveShiftLong:           6.0,
			curveShiftShort:          0.0,
			imbalancePrice:           35.0,
			imbalanceVolume:          500.0,
			ratesImport:              10,
			ratesExport:              -10,
			expectedControlComponent: activeControlComponent(600),
		},
		{
			name:                     "Test blank curves - don't charge even if prices are very negative",
			t:                        mustParseTime("2023-09-12T23:10:00+01:00"),
			soe:                      0.0,
			chargeCurve:              cartesian.Curve{},
			dischargeCurve:           cartesian.Curve{},
			curveShiftLong:           0.0,
			curveShiftShort:          0.0,
			imbalancePrice:           -999,
			imbalanceVolume:          0,
			ratesImport:              10,
			ratesExport:              -10,
			expectedControlComponent: controlComponent{},
		},
		{
			name:                     "Test blank curves - don't discharge even if prices are very high",
			t:                        mustParseTime("2023-09-12T23:10:00+01:00"),
			soe:                      200.0,
			chargeCurve:              cartesian.Curve{},
			dischargeCurve:           cartesian.Curve{},
			curveShiftLong:           0.0,
			curveShiftShort:          0.0,
			imbalancePrice:           999,
			imbalanceVolume:          0,
			ratesImport:              10,
			ratesExport:              -10,
			expectedControlComponent: controlComponent{},
		},
	}
	for _, subTest := range subTests {
		test.Run(subTest.name, func(t *testing.T) {

			// update the nivChasePeriods config for this subtest
			for i := range nivChasePeriods {
				nivChasePeriods[i].Niv.ChargeCurve = subTest.chargeCurve
				nivChasePeriods[i].Niv.DischargeCurve = subTest.dischargeCurve
				nivChasePeriods[i].Niv.CurveShiftLong = subTest.curveShiftLong
				nivChasePeriods[i].Niv.CurveShiftShort = subTest.curveShiftShort
			}

			component := nivChase(
				subTest.t,
				nivChasePeriods,
				subTest.soe,
				0.85,
				subTest.ratesImport,
				subTest.ratesExport,
				&MockImbalancePricer{
					price:  subTest.imbalancePrice,
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

func TestPredictImbalance(test *testing.T) {

	type subTest struct {
		name                  string
		t                     time.Time
		nivPredictionConfig   config.NivPredictionConfig
		modoImbalancePrice    float64
		modoImbalanceVolume   float64
		modoImbalanceDataTime time.Time
		expectedPrice         float64
		expectedVolume        float64
		expectedOK            bool
	}

	nivPredictionConfig := config.NivPredictionConfig{
		WhenShort: config.NivPredictionDirectionConfig{
			AllowPrediction: true,
			VolumeCutoff:    200,
			TimeCutoffSecs:  60 * 15,
		},
		WhenLong: config.NivPredictionDirectionConfig{
			AllowPrediction: true,
			VolumeCutoff:    3,
			TimeCutoffSecs:  60 * 15,
		},
	}

	subTests := []subTest{
		{
			name:                  "Don't trust Modo data for the first 10mins of the SP - 1",
			t:                     mustParseTime("2023-09-12T23:00:00+01:00"),
			nivPredictionConfig:   nivPredictionConfig,
			modoImbalancePrice:    -10,
			modoImbalanceVolume:   -11,
			modoImbalanceDataTime: mustParseTime("2023-09-12T23:00:00+01:00"),
			expectedPrice:         0.0,
			expectedVolume:        0.0,
			expectedOK:            false,
		},
		{
			name:                  "Don't trust Modo data for the first 10mins of the SP - 2",
			t:                     mustParseTime("2023-09-12T23:09:59+01:00"),
			nivPredictionConfig:   nivPredictionConfig,
			modoImbalancePrice:    -10,
			modoImbalanceVolume:   -11,
			modoImbalanceDataTime: mustParseTime("2023-09-12T23:00:00+01:00"),
			expectedPrice:         0.0,
			expectedVolume:        0.0,
			expectedOK:            false,
		},
		{
			name:                  "Trust Modo data after the first 10mins of the SP - 1",
			t:                     mustParseTime("2023-09-12T23:10:00+01:00"),
			nivPredictionConfig:   nivPredictionConfig,
			modoImbalancePrice:    5,
			modoImbalanceVolume:   6,
			modoImbalanceDataTime: mustParseTime("2023-09-12T23:00:00+01:00"),
			expectedPrice:         5,
			expectedVolume:        6,
			expectedOK:            true,
		},
		{
			name:                  "Trust Modo data after the first 10mins of the SP - 2",
			t:                     mustParseTime("2023-09-12T23:29:59+01:00"),
			nivPredictionConfig:   nivPredictionConfig,
			modoImbalancePrice:    5,
			modoImbalanceVolume:   6,
			modoImbalanceDataTime: mustParseTime("2023-09-12T23:00:00+01:00"),
			expectedPrice:         5,
			expectedVolume:        6,
			expectedOK:            true,
		},
		{
			name:                  "Allow prediction using previous SP data for first 15mins - 1",
			t:                     mustParseTime("2023-09-12T23:00:01+01:00"),
			nivPredictionConfig:   nivPredictionConfig,
			modoImbalancePrice:    -10,
			modoImbalanceVolume:   -11,
			modoImbalanceDataTime: mustParseTime("2023-09-12T22:30:00+01:00"),
			expectedPrice:         -10,
			expectedVolume:        -11,
			expectedOK:            true,
		},
		{
			name:                  "Allow prediction using previous SP data for first 15mins - 2",
			t:                     mustParseTime("2023-09-12T23:14:59+01:00"),
			nivPredictionConfig:   nivPredictionConfig,
			modoImbalancePrice:    -10,
			modoImbalanceVolume:   -11,
			modoImbalanceDataTime: mustParseTime("2023-09-12T22:30:00+01:00"),
			expectedPrice:         -10,
			expectedVolume:        -11,
			expectedOK:            true,
		},
		{
			name:                  "Don't allow prediction using previous SP data after the first 15mins",
			t:                     mustParseTime("2023-09-12T23:15:00+01:00"),
			nivPredictionConfig:   nivPredictionConfig,
			modoImbalancePrice:    -10,
			modoImbalanceVolume:   -11,
			modoImbalanceDataTime: mustParseTime("2023-09-12T22:30:00+01:00"),
			expectedPrice:         0,
			expectedVolume:        0,
			expectedOK:            false,
		},
		{
			name:                  "Don't allow prediction when imbalance volume is smaller than cutoff when short",
			t:                     mustParseTime("2023-09-12T23:05:00+01:00"),
			nivPredictionConfig:   nivPredictionConfig,
			modoImbalancePrice:    10,
			modoImbalanceVolume:   11,
			modoImbalanceDataTime: mustParseTime("2023-09-12T22:30:00+01:00"),
			expectedPrice:         0,
			expectedVolume:        0,
			expectedOK:            false,
		},
		{
			name:                  "Allow prediction when imbalance volume is greater than cutoff when short",
			t:                     mustParseTime("2023-09-12T23:05:00+01:00"),
			nivPredictionConfig:   nivPredictionConfig,
			modoImbalancePrice:    10,
			modoImbalanceVolume:   205,
			modoImbalanceDataTime: mustParseTime("2023-09-12T22:30:00+01:00"),
			expectedPrice:         10,
			expectedVolume:        205,
			expectedOK:            true,
		},
		{
			name:                  "Don't allow prediction when imbalance volume is smaller than cutoff when long",
			t:                     mustParseTime("2023-09-12T23:05:00+01:00"),
			nivPredictionConfig:   nivPredictionConfig,
			modoImbalancePrice:    10,
			modoImbalanceVolume:   -2,
			modoImbalanceDataTime: mustParseTime("2023-09-12T22:30:00+01:00"),
			expectedPrice:         0,
			expectedVolume:        0,
			expectedOK:            false,
		},
		{
			name:                  "Allow prediction when imbalance volume is greater than cutoff when long",
			t:                     mustParseTime("2023-09-12T23:05:00+01:00"),
			nivPredictionConfig:   nivPredictionConfig,
			modoImbalancePrice:    10,
			modoImbalanceVolume:   -20,
			modoImbalanceDataTime: mustParseTime("2023-09-12T22:30:00+01:00"),
			expectedPrice:         10,
			expectedVolume:        -20,
			expectedOK:            true,
		},
	}
	for _, subTest := range subTests {
		test.Run(subTest.name, func(t *testing.T) {

			price, volume, ok := predictImbalance(
				subTest.t,
				subTest.nivPredictionConfig,
				&MockImbalancePricer{
					price:  subTest.modoImbalancePrice,
					volume: subTest.modoImbalanceVolume,
					time:   subTest.modoImbalanceDataTime,
				},
			)

			if price != subTest.expectedPrice || volume != subTest.expectedVolume || ok != subTest.expectedOK {
				t.Errorf("got %f, %f, %t, expected %f, %f, %t", price, volume, ok, subTest.expectedPrice, subTest.expectedVolume, subTest.expectedOK)
			}
		})
	}

}

func componentsEquivalent(c1, c2 controlComponent) bool {
	if c1.isActive != c2.isActive {
		return false
	}
	if !c1.isActive {
		return true
	}
	if c1.controlPoint != c2.controlPoint {
		return false
	}
	if !almostEqual(c1.targetPower, c2.targetPower, 0.1) {
		return false
	}
	if c1.name != c2.name {
		return false
	}
	return true
}

func activeControlComponent(power float64) controlComponent {
	return controlComponent{
		name:         "niv_chase",
		isActive:     true,
		targetPower:  power,
		controlPoint: controlPointBess,
	}
}
