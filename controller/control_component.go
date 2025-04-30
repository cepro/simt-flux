package controller

// controlComponent represents the output of some control mode - e.g. export avoidance or NIV chasing etc
type controlComponent struct {
	name         string          // Friendly name of this component for debug logging
	status       componentStatus // Determines if this component is active, and if lower-priority components are allowed to alter the target power
	targetPower  float64         // The power associated with this control component
	controlPoint controlPoint    // The point where the targetPower should be applied
}

// controlPoint indicates where a target power level should be applied - i.e. at the BESS inverters or at the microgrid site
type componentStatus string

const (
	componentStatusInactive                 componentStatus = "componentStatusInactive"                 // componentStatusInactive indicates that the control component is not active and can be ignored
	componentStatusActiveGreedy             componentStatus = "componentStatusActiveGreedy"             // componentStatusActiveGreedy indicates that the control component is active and, no lower priority components should change it's target power
	componentStatusActiveAllowMoreCharge    componentStatus = "componentStatusActiveAllowMoreCharge"    // componentStatusActiveAllowMoreCharge means that the control component has an action it wants to do, but lower priority components may increase the rate of charge beyond this components target power
	componentStatusActiveAllowMoreDischarge componentStatus = "componentStatusActiveAllowMoreDischarge" // componentStatusActiveAllowMoreDischarge means that the control component has an action it wants to do, but lower priority components may increase the rate of discharge beyond this components target power
)

// controlPoint indicates where a target power level should be applied - i.e. at the BESS inverters or at the microgrid site
type controlPoint string

const (
	controlPointUnknown controlPoint = "controlPointUnknown" // controlPointUnknown indicates a default value that is not valid to use
	controlPointBess    controlPoint = "controlPointBess"    // controlPointBess indicates that a target power should be aimed for at the BESS inverters
	controlPointSite    controlPoint = "controlPointSite"    // controlPointSite indicates that a target power should be aimed for at the microgrid site boundary
)

// INACTIVE_CONTROL_COMPONENT is a pre-defined control component that does nothing as it's status is inactive.
var INACTIVE_CONTROL_COMPONENT = controlComponent{
	status: componentStatusInactive,
}
