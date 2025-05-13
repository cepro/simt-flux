package controller

import (
	"time"

	timeutils "github.com/cepro/besscontroller/time_utils"
)

// exportAvoidance returns the control component for avoiding microgrid boundary exports.
func exportAvoidance(t time.Time, exportAvoidancePeriods []timeutils.DayedPeriod, sitePower, lastTargetPower float64) controlComponent {

	_, exportAvoidancePeriod := findDayedPeriodContainingTime(t, exportAvoidancePeriods)
	if exportAvoidancePeriod == nil {
		return INACTIVE_CONTROL_COMPONENT
	}

	exportAvoidancePower := sitePower + lastTargetPower
	if exportAvoidancePower > 0 {
		return INACTIVE_CONTROL_COMPONENT
	}

	return controlComponent{
		name:         "export_avoidance",
		status:       componentStatusActiveAllowMoreCharge,
		targetPower:  0, // Target zero power at the site boundary
		controlPoint: controlPointSite,
	}
}
