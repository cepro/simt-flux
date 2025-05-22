package controller

import (
	"math"
	"time"
)

// This file contains utilities to help with testing

type MockImbalancePricer struct {
	price  float64
	volume float64
	time   time.Time
}

func (m *MockImbalancePricer) ImbalancePrice() (float64, time.Time) {
	return m.price, m.time
}

func (m *MockImbalancePricer) ImbalanceVolume() (float64, time.Time) {
	return m.volume, m.time
}

// almostEqual compares two floats, allowing for the given tolerance
func almostEqual(a, b, tolerance float64) bool {
	if a == b {
		// This is to support infinite float values
		return true
	}

	diff := math.Abs(a - b)
	return diff < tolerance
}

// mustParseTime returns the time.Time associated with the given string or panics.
func mustParseTime(str string) time.Time {
	time, err := time.Parse(time.RFC3339, str)
	if err != nil {
		panic(err)
	}
	return time
}

// componentsEquivalent returns true if c1 and c2 are equivalent.
func componentsEquivalent(c1, c2 controlComponent) bool {

	if c1.name != c2.name {
		return false
	}
	tolerance := 0.1 // 0.1kW

	if !float64PointersNearlyEqual(c1.targetPower, c2.targetPower, tolerance) {
		return false
	}

	if !float64PointersNearlyEqual(c1.minTargetPower, c2.minTargetPower, tolerance) {
		return false
	}

	if !float64PointersNearlyEqual(c1.maxTargetPower, c2.maxTargetPower, tolerance) {
		return false
	}

	return true
}

// float64PointersNearlyEqual returns true if the two float64 pointers are either both nil or both point to nearly the same value
func float64PointersNearlyEqual(p1, p2 *float64, tolerance float64) bool {
	if (p1 == nil) != (p2 == nil) {
		return false
	}
	if p1 != nil {
		return nearlyEqual(*p1, *p2, tolerance)
	}
	return true
}

func nearlyEqual(f1, f2, tolerance float64) bool {

	// Handle infinities
	if math.IsInf(f1, 0) || math.IsInf(f2, 0) {
		return f1 == f2
	}

	// Handle NaN
	if math.IsNaN(f1) || math.IsNaN(f2) {
		return false // NaN is not equal to anything, even itself
	}

	return math.Abs(f1-f2) <= tolerance
}
