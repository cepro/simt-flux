package controller

// controlPoint indicates where a target power level should be applied - i.e. at the BESS inverters or at the microgrid site
type controlPoint int64

const (
	controlPointUnknown controlPoint = iota // controlPointUnknown indicates a default value that is not valid to use
	controlPointBess                        // controlPointBess indicates that a target power should be aimed for at the BESS inverters
	controlPointSite                        // controlPointSite indicates that a target power should be aimed for at the microgrid site boundary
)
