package controller

import (
	"math"
	"time"

	"github.com/cepro/besscontroller/axle"
	"golang.org/x/exp/slog"
)

// axleSchedule returns the control component for following any Axle schedules
func axleSchedule(t time.Time, schedule axle.Schedule) controlComponent {
	action := schedule.FirstActionAt(t)
	if action == nil {
		return controlComponent{}
	}

	controlComponentName := "axle_schedule"

	if action.ActionType == "charge_max" {
		return controlComponent{
			name:         controlComponentName,
			isActive:     true,
			targetPower:  math.Inf(-1), // ask for infinite charging and allow the limits to be applied as they may
			controlPoint: controlPointBess,
		}
	} else if action.ActionType == "discharge_max" {
		return controlComponent{
			name:         controlComponentName,
			isActive:     true,
			targetPower:  math.Inf(1), // ask for infinite discharging and allow the limits to be applied as they may
			controlPoint: controlPointBess,
		}
	} else if action.ActionType == "do_nothing" {
		return controlComponent{
			name:         controlComponentName,
			isActive:     true,
			targetPower:  0,
			controlPoint: controlPointBess,
		}
	}

	slog.Error("Unknown action type from Axle", "action_type", action.ActionType)
	return controlComponent{}
}
