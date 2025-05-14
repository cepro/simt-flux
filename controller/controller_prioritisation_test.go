package controller

import (
	"testing"
)

// newTestController creates a mock controller with very generous limits (we don't want to test the limits here)
func newTestController() *Controller {
	c := New(Config{
		BessSoeMin:              0,
		BessSoeMax:              9999,
		BessChargePowerLimit:    9999,
		BessDischargePowerLimit: 9999,
		SiteImportPowerLimit:    9999,
		SiteExportPowerLimit:    9999,
	})
	c.bessSoe.set(5000) // set the SoE to middling so that the SoE doesn't form part of the constraints
	return c
}

func TestPrioritiseControlComponents_NoActiveComponents(t *testing.T) {

	// Test with empty component list
	components := []controlComponent{}
	action := newTestController().prioritiseControlComponents(components)

	// Verify default "idle" component is used
	if action.activeComponentNames != "idle" {
		t.Errorf("Expected 'idle' active component, got %s", action.activeComponentNames)
	}
	if action.bessTargetPower != 0 {
		t.Errorf("Expected 0 power, got %f", action.bessTargetPower)
	}

	// Test with inactive components only
	components = []controlComponent{
		{
			name:           "component1",
			targetPower:    nil,
			minTargetPower: nil,
			maxTargetPower: nil,
		},
		{
			name:           "component2",
			targetPower:    nil,
			minTargetPower: nil,
			maxTargetPower: nil,
		},
	}

	action = newTestController().prioritiseControlComponents(components)

	// Verify default "idle" component is still used
	if action.activeComponentNames != "idle" {
		t.Errorf("Expected 'idle' active component, got %s", action.activeComponentNames)
	}
	if action.bessTargetPower != 0 {
		t.Errorf("Expected 0 power, got %f", action.bessTargetPower)
	}
}

func TestPrioritiseControlComponents_GreedyComponent(t *testing.T) {

	components := []controlComponent{
		{
			name:           "greedy",
			targetPower:    pointerToFloat64(100),
			minTargetPower: pointerToFloat64(100),
			maxTargetPower: pointerToFloat64(100),
		},
		{
			name:           "lower_priority",
			targetPower:    pointerToFloat64(200),
			minTargetPower: nil,
			maxTargetPower: nil,
		},
	}

	action := newTestController().prioritiseControlComponents(components)

	if action.bessTargetPower != 100 {
		t.Errorf("Expected 100 power, got %f", action.bessTargetPower)
	}
}

func TestPrioritiseControlComponents_AllowMoreCharge(t *testing.T) {

	components := []controlComponent{
		{
			name:           "allow_more_charge",
			targetPower:    pointerToFloat64(-50),
			minTargetPower: nil,                   // allow more negative (faster charge rate)
			maxTargetPower: pointerToFloat64(-50), // don't allow slower charging or discharging
		},
		{
			name:           "discharge_more", // discharge should be ignored
			targetPower:    pointerToFloat64(200),
			minTargetPower: nil,
			maxTargetPower: nil,
		},
		{
			name:           "charge_more", // Should be allowed since it charges more
			targetPower:    pointerToFloat64(-100),
			minTargetPower: nil,
			maxTargetPower: pointerToFloat64(-100),
		},
		{
			name:           "charge_less", // Should be ignored since it charges less
			targetPower:    pointerToFloat64(-75),
			minTargetPower: nil,
			maxTargetPower: nil,
		},
	}

	action := newTestController().prioritiseControlComponents(components)

	if action.bessTargetPower != -100 {
		t.Errorf("Expected -100 power, got %f", action.bessTargetPower)
	}
}

func TestPrioritiseControlComponents_AllowMoreDischarge(t *testing.T) {

	components := []controlComponent{
		{
			name:           "inactive",
			targetPower:    nil,
			minTargetPower: nil,
			maxTargetPower: nil,
		},
		{
			name:           "allow_more_discharge",
			targetPower:    pointerToFloat64(50),
			minTargetPower: pointerToFloat64(50), // don't allow slower discharging or charging
			maxTargetPower: nil,                  // allow more positive (faster discharge rate)
		},
		{
			name:           "charge_more", // charge should be ignored
			targetPower:    pointerToFloat64(-100),
			minTargetPower: nil,
			maxTargetPower: pointerToFloat64(-100),
		},
		{
			name:           "discharge_greedy", // Should be allowed since it discharges more
			targetPower:    pointerToFloat64(100),
			minTargetPower: pointerToFloat64(100),
			maxTargetPower: pointerToFloat64(100),
		},
		{
			name:           "discharge_more", // Should be ignored since the previous component was greedy
			targetPower:    pointerToFloat64(150),
			minTargetPower: pointerToFloat64(150),
			maxTargetPower: nil,
		},
	}

	action := newTestController().prioritiseControlComponents(components)

	if action.bessTargetPower != 100 {
		t.Errorf("Expected 100 power, got %f", action.bessTargetPower)
	}
}
