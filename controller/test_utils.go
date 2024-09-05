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

// componentsEquivalent returns true if c1 and c2 are equivalent
func componentsEquivalent(c1, c2 controlComponent) bool {
	if c1.isActive != c2.isActive {
		return false
	}
	if !c1.isActive {
		return true
	}
	if c1.controlPoint != c2.controlPoint {
		return false
	}
	if !almostEqual(c1.targetPower, c2.targetPower, 0.1) {
		return false
	}
	if c1.name != c2.name {
		return false
	}
	return true
}
