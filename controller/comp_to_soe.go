package controller

import (
	"time"

	"github.com/cepro/besscontroller/config"
)

// chargeToSoe returns the control component for charging the battery to a minimum SoE.
func chargeToSoe(t time.Time, configs []config.DayedPeriodWithSoe, bessSoe, chargeEfficiency float64) controlComponent {

	conf, absPeriod := findPeriodicalConfigForTime(t, configs)
	if conf == nil {
		return INACTIVE_CONTROL_COMPONENT
	}

	targetSoe := conf.Soe
	endOfCharge := absPeriod.End

	// charge the battery to reach the minimum target SoE at the end of the period. If the battery is already charged to the minimum level then do nothing.
	energyToCharge := (targetSoe - bessSoe) / chargeEfficiency
	if energyToCharge <= 0 {
		return INACTIVE_CONTROL_COMPONENT
	}

	durationToRecharge := endOfCharge.Sub(t)
	chargePower := -energyToCharge / durationToRecharge.Hours()
	if chargePower >= 0 {
		return INACTIVE_CONTROL_COMPONENT
	}

	return controlComponent{
		name:         "charge_to_soe",
		status:       componentStatusActiveAllowMoreCharge,
		targetPower:  chargePower,
		controlPoint: controlPointBess,
	}
}

// dischargeToSoe returns the control component for discharging the battery to a pre-defined state of energy.
func dischargeToSoe(t time.Time, configs []config.DayedPeriodWithSoe, bessSoe, dischargeEfficiency float64) controlComponent {

	conf, absPeriod := findPeriodicalConfigForTime(t, configs)
	if conf == nil {
		return INACTIVE_CONTROL_COMPONENT
	}

	targetSoe := conf.Soe
	endOfDischarge := absPeriod.End

	// discharge the battery to reach the target SoE at the end of the period. If the battery is already discharged to the target level, or below then do nothing.
	energyToDischarge := (bessSoe - targetSoe) * dischargeEfficiency
	if energyToDischarge <= 0 {
		return INACTIVE_CONTROL_COMPONENT
	}

	durationToDischarge := endOfDischarge.Sub(t)
	dichargePower := energyToDischarge / durationToDischarge.Hours()
	if dichargePower <= 0 {
		return INACTIVE_CONTROL_COMPONENT
	}

	return controlComponent{
		name:         "discharge_to_soe",
		status:       componentStatusActiveAllowMoreDischarge,
		targetPower:  dichargePower,
		controlPoint: controlPointBess,
	}
}
