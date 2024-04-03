package controller

import (
	"fmt"
	"time"

	"github.com/cepro/besscontroller/cartesian"
	"github.com/cepro/besscontroller/config"
	timeutils "github.com/cepro/besscontroller/time_utils"
	"golang.org/x/exp/slog"
)

const (
	thirtyMins = time.Minute * 30
)

// nivChase returns the control component for NIV chasing, using the Modo imbalance price calculation.
func nivChase(
	t time.Time,
	nivChasePeriods []config.DayedPeriodWithNIV,
	soe,
	chargeEfficiency,
	chargesImport,
	chargesExport float64,
	modoClient imbalancePricer,
) controlComponent {

	logger := slog.Default()

	periodWithNiv := periodWithNivContainingTime(t, nivChasePeriods)
	if periodWithNiv == nil {
		return controlComponent{isActive: false}
	}

	currentSP := timeutils.FloorHH(t)
	timeIntoSP := t.Sub(currentSP)
	timeLeftOfSP := thirtyMins - timeIntoSP

	// Sanity check:
	if timeLeftOfSP > thirtyMins || timeLeftOfSP < 0 {
		panic(fmt.Sprintf("Time left of SP is invalid: %v", timeLeftOfSP))
	}

	// Check if we have default pricing for this period
	defaultImbalancePrice, foundDefaultImbalancePrice := config.FirstTimedCharges(t, periodWithNiv.Niv.DefaultPricing)

	// We only trust the imbalance price calcualation 10 minutes into the SP - unless a default was provided
	if (timeIntoSP < time.Minute*10) && !foundDefaultImbalancePrice {
		logger.Info("Too soon into settlement period to NIV chase")
		return controlComponent{isActive: false}
	}

	modoImbalancePrice, modoImbalancePriceSP := modoClient.ImbalancePrice()
	foundModoImbalancePrice := currentSP.Equal(modoImbalancePriceSP)

	modoImbalanceVolume, modoImbalanceVolumeSP := modoClient.ImbalanceVolume()
	foundModoImbalanceVolume := currentSP.Equal(modoImbalanceVolumeSP)

	// Make sure we have a price prediction for the current SP - sometimes Modo can take a while to generate a calculation, and in the mean-time will continue to
	// publish the price for the previous SP.
	if !foundModoImbalancePrice && !foundDefaultImbalancePrice {
		logger.Info("Cannot NIV chase: modo imbalance price is for the wrong settlement period", "current_settlement_period", currentSP, "price_settlement_period", modoImbalancePriceSP)
		return controlComponent{isActive: false}
	}

	var imbalancePrice float64
	if !foundModoImbalancePrice && foundDefaultImbalancePrice {
		logger.Info("Using default imbalance price", "default_imbalance_price", defaultImbalancePrice)
		imbalancePrice = defaultImbalancePrice
	} else {
		imbalancePrice = modoImbalancePrice
	}

	chargePrice := imbalancePrice + chargesImport
	dischargePrice := imbalancePrice - chargesExport

	// Shift the curves depending on if the system is long or short - this is achieved in practice by adjusting the price input into the curve
	shift := 0.0
	shiftedChargePrice := chargePrice
	shiftedDischargePrice := dischargePrice
	imbalanceDirectionStr := "unknown"
	if foundModoImbalanceVolume {
		isLong := modoImbalanceVolume < 0
		if isLong {
			shift = -periodWithNiv.Niv.CurveShiftLong
			imbalanceDirectionStr = "long"
		} else {
			shift = periodWithNiv.Niv.CurveShiftShort
			imbalanceDirectionStr = "short"
		}
	}
	shiftedChargePrice += shift
	shiftedDischargePrice += shift

	chargeDistance := periodWithNiv.Niv.ChargeCurve.VerticalDistance(cartesian.Point{X: shiftedChargePrice, Y: soe})
	dischargeDistance := periodWithNiv.Niv.DischargeCurve.VerticalDistance(cartesian.Point{X: shiftedDischargePrice, Y: soe})
	energyDelta := 0.0

	if chargeDistance > 0 {
		energyDelta = -chargeDistance / chargeEfficiency
	} else if dischargeDistance < 0 {
		energyDelta = -dischargeDistance
	}

	targetPower := energyDelta / timeLeftOfSP.Hours()

	logger.Info(
		"NIV chasing debug",
		"target_energy_delta", energyDelta,
		"target_power", targetPower,
		"time_left", timeLeftOfSP.Hours(),
		"charge_price", chargePrice,
		"discharge_price", dischargePrice,
		"imbalance_direction", imbalanceDirectionStr,
		"shifted_charge_price", shiftedChargePrice,
		"shifted_discharge_price", shiftedDischargePrice,
		"charge_distance", chargeDistance,
		"discharge_distance", dischargeDistance,
	)

	// Battery power constraints are applied upstream...
	return controlComponent{
		name:         "niv_chase",
		isActive:     (targetPower > 0) || (targetPower < 0),
		targetPower:  targetPower,
		controlPoint: controlPointBess,
	}
}
