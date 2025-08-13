package controller

import "fmt"

// controlComponent represents the output of some control mode - e.g. export avoidance or NIV chasing etc
type controlComponent struct {
	name string // Friendly name of this component for debug logging

	// These fields are nullable - a nil value means that the control component does not have any preference.
	// Positive values are discharges, negative values are charges.
	targetPower    *float64 // The power that this control component wants the battery to do, or nil if it has no preference
	minTargetPower *float64 // The minimum power that any lower-priority component are allowed to do, or nil if there is no restriction
	maxTargetPower *float64 // the maximum power that a lower-priority component is allowed to do, or nil if there is no restriction
}

// isActive returns true if the control component has any active instructions
func (c *controlComponent) isActive() bool {
	return c.targetPower != nil || c.minTargetPower != nil || c.maxTargetPower != nil
}

func (c *controlComponent) str() string {
	return fmt.Sprintf("'%s'/%s/%s/%s", c.name, strForPointerToFloat64(c.targetPower), strForPointerToFloat64(c.minTargetPower), strForPointerToFloat64(c.maxTargetPower))
}

// strForPointerToFloat64 returns a string representing the *float64
func strForPointerToFloat64(p *float64) string {
	s := "nil"
	if p != nil {
		s = fmt.Sprintf("%.2f", *p)
	}
	return s
}

// chargingControlComponentThatAllowsMoreCharge returns a control component which charges at the given power level (which must be negative to indicate a charge), and
// also allows any lower-priority components that wish to charge even faster to do so. It doesn't allow lower-priority components to charge at a slower rate or to discharge.
func chargingControlComponentThatAllowsMoreCharge(name string, power float64) controlComponent {

	if power > 0 {
		panic("charging powers are negative")
	}

	return controlComponent{
		name:           name,
		targetPower:    &power,
		minTargetPower: nil,    // allow the target power to become more negative (i.e. faster charge)
		maxTargetPower: &power, // prevent the target power from becoming less negative (i.e. slower charge)
	}
}

// dischargingControlComponentThatAllowsMoreDischarge returns a control component which discharges at the given power level (which must be positive to indicate a discharge), and
// also allows any lower-priority components that wish to discharge even faster to do so. It doesn't allow lower-priority components to discharge at a slower rate or to charge.
func dischargingControlComponentThatAllowsMoreDischarge(name string, power float64) controlComponent {

	if power < 0 {
		panic("discharging powers are positive")
	}

	return controlComponent{
		name:           name,
		targetPower:    &power,
		minTargetPower: &power, // prevent the target power from becoming less positive (i.e. slower discharge)
		maxTargetPower: nil,    // allow the target power to become more positive (i.e. faster discharge)
	}
}

// INACTIVE_CONTROL_COMPONENT is a pre-defined control component that does nothing: no target power or limits are specified.
var INACTIVE_CONTROL_COMPONENT = controlComponent{
	name:           "",
	targetPower:    nil,
	minTargetPower: nil,
	maxTargetPower: nil,
}
