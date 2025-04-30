package controller

import (
	"time"

	"github.com/cepro/besscontroller/config"
	timeutils "github.com/cepro/besscontroller/time_utils"
)

// importAvoidanceWhenShort returns control component for avoiding site imports, based on imbalance status
func importAvoidanceWhenShort(t time.Time, configs []config.ImportAvoidanceWhenShortConfig, sitePower, lastTargetPower float64, modoClient imbalancePricer) controlComponent {

	conf, _ := findPeriodicalConfigForTime(t, configs)
	if conf == nil {
		return INACTIVE_CONTROL_COMPONENT
	}

	_, imbalanceVolume, gotPrediction := predictImbalance(
		t,
		config.NivPredictionConfig{
			WhenShort: conf.ShortPrediction,
			// We are only interested in the case where the system is short, so don't allow predictions for long
			WhenLong: config.NivPredictionDirectionConfig{AllowPrediction: false},
		},
		modoClient,
	)
	if !gotPrediction {
		// We don't have any pricing data available, so do nothing
		return INACTIVE_CONTROL_COMPONENT
	}

	if imbalanceVolume <= 0 {
		// We aren't short, so do nothing
		return INACTIVE_CONTROL_COMPONENT
	}

	return importAvoidanceHelper(sitePower, lastTargetPower, "import_avoidance_when_short", true)
}

// basicImportAvoidance returns the control component for avoiding microgrid boundary imports.
func basicImportAvoidance(t time.Time, importAvoidancePeriods []timeutils.DayedPeriod, sitePower, lastTargetPower float64) controlComponent {

	_, importAvoidancePeriod := findDayedPeriodContainingTime(t, importAvoidancePeriods)
	if importAvoidancePeriod == nil {
		return INACTIVE_CONTROL_COMPONENT
	}

	return importAvoidanceHelper(sitePower, lastTargetPower, "import_avoidance", true)
}

// importAvoidanceHelper generates the control component for an import avoidance action.
// Import avoidance is a strategy that is used by a few different control modes so this is a conveninence function to help create the correct control component.
func importAvoidanceHelper(sitePower, lastTargetPower float64, controlComponentName string, allowMoreDischarge bool) controlComponent {
	importAvoidancePower := sitePower + lastTargetPower
	if importAvoidancePower < 0 {
		return INACTIVE_CONTROL_COMPONENT // there is nothing to do as the site is not importing
	}

	status := componentStatusActiveGreedy
	if allowMoreDischarge {
		status = componentStatusActiveAllowMoreDischarge
	}

	return controlComponent{
		name:         controlComponentName,
		status:       status,
		targetPower:  0, // Target zero power at the site boundary
		controlPoint: controlPointSite,
	}
}
