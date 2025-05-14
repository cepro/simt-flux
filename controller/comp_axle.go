package controller

import (
	"math"
	"time"

	"github.com/cepro/besscontroller/axle"
	"golang.org/x/exp/slog"
)

// axleSchedule returns the control component for following any Axle schedules
func axleSchedule(t time.Time, schedule axle.Schedule, sitePower, lastTargetPower float64) controlComponent {
	scheduleItem := schedule.FirstItemAt(t)
	if scheduleItem == nil {
		return INACTIVE_CONTROL_COMPONENT
	}

	if scheduleItem.Action == "charge_max" {
		return controlComponent{
			name:           "axle_schedule.charge_max",
			targetPower:    pointerToFloat64(math.Inf(-1)), // ask for infinite charging and allow the limits to be applied as they may
			minTargetPower: pointerToFloat64(math.Inf(-1)),
			maxTargetPower: pointerToFloat64(math.Inf(-1)),
		}
	} else if scheduleItem.Action == "discharge_max" {
		return controlComponent{
			name:           "axle_schedule.discharge_max",
			targetPower:    pointerToFloat64(math.Inf(1)), // ask for infinite charging and allow the limits to be applied as they may
			minTargetPower: pointerToFloat64(math.Inf(1)),
			maxTargetPower: pointerToFloat64(math.Inf(1)),
		}
	} else if scheduleItem.Action == "avoid_import" {
		return importAvoidanceHelper(sitePower, lastTargetPower, "axle_schedule.avoid_import", true)
	} else if scheduleItem.Action == "avoid_export" {
		return exportAvoidanceHelper(sitePower, lastTargetPower, "axle_schedule.avoid_export", true)
	}

	slog.Error("Unknown action type from Axle", "action_type", scheduleItem.Action)
	return INACTIVE_CONTROL_COMPONENT
}
