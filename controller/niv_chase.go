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

// nivChase returns the power level that should be applied to the battery for NIV chasing, and a boolean indicating if NIV chasing is active.
func nivChase(
	t time.Time,
	nivChasePeriods []timeutils.ClockTimePeriod,
	chargeCurve, dischargeCurve cartesian.Curve,
	soe,
	chargeEfficiency,
	duosChargeImport,
	duosChargeExport float64,
	modoClient imbalancePricer,
) (float64, bool) {

	logger := slog.Default()

	nivChasePeriod := periodContainingTime(t, nivChasePeriods)
	if nivChasePeriod == nil {
		return 0, false
	}

	currentSP := timeutils.FloorHH(t)
	timeIntoSP := t.Sub(currentSP)
	timeLeftOfSP := thirtyMins - timeIntoSP

	// Sanity check:
	if timeLeftOfSP > thirtyMins || timeLeftOfSP < 0 {
		panic(fmt.Sprintf("Time left of SP is invalid: %v", timeLeftOfSP))
	}

	// We only trust the imbalance price calcualation 10 minutes into the SP
	if timeIntoSP < time.Minute*10 {
		logger.Info("Too soon into settlement period to NIV chase")
		return 0, false
	}

	imbalancePrice, imbalancePriceSP := modoClient.ImbalancePrice()

	// Make sure we have a price prediction for the current SP - sometimes Modo can take a while to generate a calculation, and in the mean-time will continue to
	// publish the price for the previous SP.
	if !currentSP.Equal(imbalancePriceSP) {
		logger.Info("Cannot NIV chase: imbalance price is for the wrong settlement period", "current_settlement_period", currentSP, "price_settlement_period", imbalancePriceSP)
		return 0, false
	}

	chargeDistance := chargeCurve.VerticalDistance(cartesian.Point{X: imbalancePrice + duosChargeImport, Y: soe})
	dischargeDistance := dischargeCurve.VerticalDistance(cartesian.Point{X: imbalancePrice - duosChargeExport, Y: soe})
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
		"charge_price", imbalancePrice+duosChargeImport,
		"discharge_price", imbalancePrice-duosChargeExport,
		"charge_distance", chargeDistance,
		"discharge_distance", dischargeDistance,
	)

	// Battery power constraints are applied upstream...

	isActive := (targetPower > 0) || (targetPower < 0)
	return targetPower, isActive
}
