package controller

import (
	"fmt"
	"time"

	"github.com/cepro/besscontroller/cartesian"
	timeutils "github.com/cepro/besscontroller/time_utils"
	"golang.org/x/exp/slog"
)

const (
	thirtyMins = time.Minute * 30
)

// nivChase returns the control component for NIV chasing, using the Modo imbalance price calculation.
func nivChase(
	t time.Time,
	nivChasePeriods []timeutils.ClockTimePeriod,
	defaultPricing []TimedCharge,
	chargeCurve, dischargeCurve cartesian.Curve,
	soe,
	chargeEfficiency,
	duosChargeImport,
	duosChargeExport float64,
	modoClient imbalancePricer,
) controlComponent {

	logger := slog.Default()

	nivChasePeriod := periodContainingTime(t, nivChasePeriods)
	if nivChasePeriod == nil {
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
	defaultImbalancePrice, foundDefaultImbalancePrice := firstTimedCharges(t, defaultPricing)

	// We only trust the imbalance price calcualation 10 minutes into the SP - unless a default was provided
	if (timeIntoSP < time.Minute*10) && !foundDefaultImbalancePrice {
		logger.Info("Too soon into settlement period to NIV chase")
		return controlComponent{isActive: false}
	}

	modoImbalancePrice, modoImbalancePriceSP := modoClient.ImbalancePrice()
	foundModoImbalancePrice := currentSP.Equal(modoImbalancePriceSP)

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

	chargePrice := imbalancePrice + duosChargeImport
	dischargePrice := imbalancePrice - duosChargeExport

	chargeDistance := chargeCurve.VerticalDistance(cartesian.Point{X: chargePrice, Y: soe})
	dischargeDistance := dischargeCurve.VerticalDistance(cartesian.Point{X: dischargePrice, Y: soe})
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
		"discharge_price", dischargeDistance,
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
