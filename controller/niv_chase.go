package controller

import (
	"fmt"
	"math"
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
	configs []config.DayedPeriodWithNIV,
	soe,
	chargeEfficiency,
	rateImport,
	rateExport float64,
	modoClient imbalancePricer,
) controlComponent {

	logger := slog.Default()

	conf, _ := findPeriodicalConfigForTime(t, configs)
	if conf == nil {
		return controlComponent{isActive: false}
	}

	imbalancePrice, imbalanceVolume, gotPrediction := predictImbalance(t, conf.Niv.Prediction, modoClient)
	if !gotPrediction {
		// Check if we have default pricing configured that we can use in lieu of the predictions
		defaultImbalancePrice, gotDefaultPrice := config.FirstTimedRate(t, conf.Niv.DefaultPricing)
		if gotDefaultPrice {
			imbalancePrice = defaultImbalancePrice
		} else {
			// We don't have any pricing data available, so do nothing
			return controlComponent{isActive: false}
		}
	}

	// Add on supplier and DUoS rates etc
	chargePrice := imbalancePrice + rateImport
	dischargePrice := imbalancePrice - rateExport

	// Shift the curves depending on if the system is long or short - this is achieved in practice by adjusting the price input into the curve
	shift := 0.0
	shiftedChargePrice := chargePrice
	shiftedDischargePrice := dischargePrice
	imbalanceDirectionStr := "unknown" // just for logging
	if imbalanceVolume < 0 {
		shift = -conf.Niv.CurveShiftLong
		imbalanceDirectionStr = "long"
	} else if imbalanceVolume > 0 {
		shift = conf.Niv.CurveShiftShort
		imbalanceDirectionStr = "short"
	} else {
		// If we don't have an imbalance volume (or it's actually 0) then don't shift in either direction
	}
	shiftedChargePrice += shift
	shiftedDischargePrice += shift

	// Lookup the charge/discharge curves to determine the power level
	chargeDistance := conf.Niv.ChargeCurve.VerticalDistance(cartesian.Point{X: shiftedChargePrice, Y: soe})
	dischargeDistance := conf.Niv.DischargeCurve.VerticalDistance(cartesian.Point{X: shiftedDischargePrice, Y: soe})
	energyDelta := 0.0

	if chargeDistance > 0 {
		energyDelta = -chargeDistance / chargeEfficiency
	} else if dischargeDistance < 0 {
		energyDelta = -dischargeDistance
	}

	currentSP := timeutils.FloorHH(t)
	timeLeftOfCurrentSP := thirtyMins - t.Sub(currentSP)
	targetPower := energyDelta / timeLeftOfCurrentSP.Hours()

	logger.Info(
		"NIV chasing debug",
		"target_energy_delta", energyDelta,
		"target_power", targetPower,
		"time_left", timeLeftOfCurrentSP.Hours(),
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

// predictImbalance returns a predition of the imbalance price and volume for this settlement period, and a boolean indicating if the
// prediction was successfull.
func predictImbalance(t time.Time, nivPredictionConfig config.NivPredictionConfig, modoClient imbalancePricer) (float64, float64, bool) {

	logger := slog.Default()

	currentSP := timeutils.FloorHH(t)
	previousSP := currentSP.Add(-thirtyMins)
	timeIntoCurrentSP := t.Sub(currentSP)
	timeLeftOfCurrentSP := thirtyMins - timeIntoCurrentSP

	// Sanity check
	if timeLeftOfCurrentSP > thirtyMins || timeLeftOfCurrentSP < 0 {
		panic(fmt.Sprintf("Time left of SP is invalid: %v", timeLeftOfCurrentSP))
	}

	// Pull the latest Modo imbalance data from the cache (actual API access is happening in the background)
	modoImbalancePrice, modoImbalancePriceSP := modoClient.ImbalancePrice()
	modoImbalanceVolume, modoImbalanceVolumeSP := modoClient.ImbalanceVolume()

	// There is a delay at the start of each SP before the Modo API updates to reflect the current SP. During the delay the
	// API serves the data for the previous SP (or potentially even older SPs if there is an issue with Modo or the
	// Elexon servers etc).
	modoDataIsForCurrentSP := currentSP.Equal(modoImbalancePriceSP) && currentSP.Equal(modoImbalanceVolumeSP)
	modoDataIsForPreviousSP := previousSP.Equal(modoImbalancePriceSP) && previousSP.Equal(modoImbalanceVolumeSP)

	if modoDataIsForCurrentSP {
		// We only trust the imbalance price calcualation 10 minutes into the SP, before then it can be a bit innacurate, so we don't act on it
		if timeIntoCurrentSP < time.Minute*10 {
			logger.Info("Too soon into settlement period to trust modo calculation")
			return 0.0, 0.0, false
		}
		// We have a valid prediction for the current SP
		return modoImbalancePrice, modoImbalanceVolume, true
	}

	// We don't have Modo data for this SP, but we may be able to use the previous SP's imbalance data as a prediction
	// for the first minutes of this SP.
	if modoDataIsForPreviousSP {
		// There is different prediction configuration depending on if the system was short or long in the previous SP.
		directionalConfig := config.NivPredictionDirectionConfig{}
		if modoImbalanceVolume > 0 {
			directionalConfig = nivPredictionConfig.WhenShort // system was short
		} else {
			directionalConfig = nivPredictionConfig.WhenLong // system was short
		}

		if !directionalConfig.AllowPrediction {
			return 0.0, 0.0, false
		}

		timeCutoff := time.Duration(directionalConfig.TimeCutoffSecs * int(time.Second))
		if timeIntoCurrentSP >= timeCutoff {
			logger.Info("Too late in settlement period to use previous SP imbalance data")
			// Only allow the use of the previous SP data for a short while - the Modo API should take over after 10mins
			return 0.0, 0.0, false
		}

		if math.Abs(modoImbalanceVolume) < directionalConfig.VolumeCutoff {
			// If the previous imbalance volume was too small then don't allow a prediction to be made as the system
			// is more likely to flip between long and short states when then the imbalance magnitude is small
			logger.Info("Imbalance volume to small to use previous SP imbalance data")
			return 0.0, 0.0, false
		}

		logger.Info("Using previous settlement periods imbalance data as predictor")
		return modoImbalancePrice, modoImbalanceVolume, true
	}

	logger.Info("Cannot NIV chase: modo imbalance price is for an old settlement period", "current_settlement_period", currentSP, "price_settlement_period", modoImbalancePriceSP, "volume_settlement_period", modoImbalanceVolumeSP)
	return 0.0, 0.0, false
}
