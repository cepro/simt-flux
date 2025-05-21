package controller

// activeConstraints provides information on which constraints were used in the calculation of the BESS power level (useful for debugging).
type activeConstraints struct {
	bessPower bool // set if the BESS inverter power rating was a limiting factor
	sitePower bool // set if the grid connection power rating was a limiting factor
	bessSoe   bool // set if the BESS SoE was a limiting factor
}

// add combines the two sets of constraints
func (a activeConstraints) add(other activeConstraints) activeConstraints {
	return activeConstraints{
		bessPower: a.bessPower || other.bessPower,
		sitePower: a.sitePower || other.sitePower,
		bessSoe:   a.bessSoe || other.bessSoe,
	}
}
