package controller

import (
	"time"

	timeutils "github.com/cepro/besscontroller/time_utils"
)

// basicExportAvoidance returns the control component for avoiding microgrid boundary exports.
func basicExportAvoidance(t time.Time, exportAvoidancePeriods []timeutils.DayedPeriod, sitePower, lastTargetPower float64) controlComponent {

	_, exportAvoidancePeriod := findDayedPeriodContainingTime(t, exportAvoidancePeriods)
	if exportAvoidancePeriod == nil {
		return INACTIVE_CONTROL_COMPONENT
	}

	return exportAvoidanceHelper(sitePower, lastTargetPower, "export_avoidance", true)
}

// exportAvoidanceHelper generates the control component for an export avoidance action.
// Export avoidance is a strategy that is used by a few different control modes so this is a conveninence function to help create the correct control component.
func exportAvoidanceHelper(sitePower, lastTargetPower float64, controlComponentName string, allowMoreCharge bool) controlComponent {

	exportAvoidancePower := sitePower + lastTargetPower
	if exportAvoidancePower > 0 {
		// In this case we don't need to tell the battery to do anything in order to achieve 'export avoidance', however, we
		// do need to limit any lower-priority components from discharging so much as to trigger an export. We do this by setting
		// the maximum BESS target power here.
		return controlComponent{
			name:           controlComponentName,
			targetPower:    nil,
			minTargetPower: nil,
			maxTargetPower: &exportAvoidancePower,
		}
	}

	// As long as the battery is charging at least `exportAvoidancePower` than we probably
	// don't mind if it charges evem more than that.
	minBessTargetPower := &exportAvoidancePower
	if allowMoreCharge {
		minBessTargetPower = nil
	}

	return controlComponent{
		name:           controlComponentName,
		targetPower:    &exportAvoidancePower,
		minTargetPower: minBessTargetPower,
		maxTargetPower: &exportAvoidancePower,
	}
}
