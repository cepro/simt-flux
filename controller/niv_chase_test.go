package controller

import (
	"testing"
	"time"

	"github.com/cepro/besscontroller/cartesian"
	timeutils "github.com/cepro/besscontroller/time_utils"
)

func TestNivChase(test *testing.T) {

	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		test.Fatalf("Could not load location: %v", err)
	}

	nivChasePeriods := []timeutils.ClockTimePeriod{
		{
			Start: timeutils.ClockTime{Hour: 23, Minute: 0, Second: 0, Location: london},
			End:   timeutils.ClockTime{Hour: 23, Minute: 59, Second: 59, Location: london},
		},
	}

	nivChargeCurve := cartesian.Curve{
		Points: []cartesian.Point{
			{X: -9999, Y: 180},
			{X: 0, Y: 180},
			{X: 20, Y: 0},
		},
	}
	nivDischargeCurve := cartesian.Curve{
		Points: []cartesian.Point{
			{X: 30, Y: 180},
			{X: 40, Y: 0},
			{X: 9999, Y: 0},
		},
	}

	type subTest struct {
		name                     string
		t                        time.Time
		soe                      float64
		curveShiftLong           float64
		curveShiftShort          float64
		imbalancePrice           float64
		imbalanceVolume          float64
		duosChargeImport         float64
		duosChargeExport         float64
		expectedControlComponent controlComponent
	}

	subTests := []subTest{
		{
			name:                     "No NIV chasing before we trust the imbalance price at 10mins into the SP - test1",
			t:                        mustParseTime("2023-09-12T23:00:00+01:00"),
			soe:                      19.0,
			curveShiftLong:           0.0,
			curveShiftShort:          0.0,
			imbalancePrice:           -99,
			imbalanceVolume:          0.0,
			expectedControlComponent: controlComponent{},
		},
		{
			name:                     "No NIV chasing before we trust the imbalance price at 10mins into the SP - test2",
			t:                        mustParseTime("2023-09-12T23:09:59+01:00"),
			soe:                      19.0,
			curveShiftLong:           0.0,
			curveShiftShort:          0.0,
			imbalancePrice:           +99,
			imbalanceVolume:          0.0,
			expectedControlComponent: controlComponent{},
		},
		{
			name:                     "Imbalance price is between the charge and discharge curves - no action",
			t:                        mustParseTime("2023-09-12T23:10:00+01:00"),
			soe:                      100.0,
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
			curveShiftLong:           5,
			curveShiftShort:          5,
			imbalancePrice:           35.0,
			imbalanceVolume:          50,
			expectedControlComponent: activeControlComponent(300.0),
		},
		{
			name:                     "Imbalance price is between the charge and discharge curves - but DUoS fees trigger export",
			t:                        mustParseTime("2023-09-12T23:10:00+01:00"),
			soe:                      200.0,
			curveShiftLong:           0.0,
			curveShiftShort:          0.0,
			imbalancePrice:           25.0,
			imbalanceVolume:          0.0,
			duosChargeImport:         5,
			duosChargeExport:         5,
			expectedControlComponent: activeControlComponent(60),
		},
	}
	for _, subTest := range subTests {
		test.Run(subTest.name, func(t *testing.T) {

			component := nivChase(
				subTest.t,
				nivChasePeriods,
				[]TimedCharge{},
				nivChargeCurve,
				nivDischargeCurve,
				subTest.curveShiftLong,
				subTest.curveShiftShort,
				subTest.soe,
				0.85,
				subTest.duosChargeImport,
				subTest.duosChargeExport,
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

func componentsEquivalent(c1, c2 controlComponent) bool {
	if c1.isActive != c1.isActive {
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
