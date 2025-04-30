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
		return controlComponent{}
	}

	if scheduleItem.Action == "charge_max" {
		return controlComponent{
			name:         "axle_schedule.charge_max",
			status:       componentStatusActiveGreedy,
			targetPower:  math.Inf(-1), // ask for infinite charging and allow the limits to be applied as they may
			controlPoint: controlPointBess,
		}
	} else if scheduleItem.Action == "discharge_max" {
		return controlComponent{
			name:         "axle_schedule.discharge_max",
			status:       componentStatusActiveGreedy,
			targetPower:  math.Inf(1), // ask for infinite discharging and allow the limits to be applied as they may
			controlPoint: controlPointBess,
		}
	} else if scheduleItem.Action == "avoid_import" {
		return controlComponent{
			name:         "axle_schedule.avoid_import",
			status:       componentStatusActiveGreedy,
			targetPower:  0, // Target zero power at the site boundary
			controlPoint: controlPointSite,
		}
	} else if scheduleItem.Action == "avoid_export" {
		return controlComponent{
			name:         "axle_schedule.avoid_export",
			status:       componentStatusActiveGreedy,
			targetPower:  0, // Target zero power at the site boundary
			controlPoint: controlPointSite,
		}
	}
	// TODO: the avoid_import and avoid_export have been a bit tricky to do properly because they need to signal to later components that they can't import/export,

	slog.Error("Unknown action type from Axle", "action_type", scheduleItem.Action)
	return controlComponent{}
}
