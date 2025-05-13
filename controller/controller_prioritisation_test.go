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
			name:         "component1",
			status:       componentStatusInactive,
			targetPower:  100,
			controlPoint: controlPointBess,
		},
		{
			name:         "component2",
			status:       componentStatusInactive,
			targetPower:  200,
			controlPoint: controlPointBess,
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
			name:         "greedy",
			status:       componentStatusActiveGreedy,
			targetPower:  100,
			controlPoint: controlPointBess,
		},
		{
			name:         "lower_priority",
			status:       componentStatusActiveGreedy,
			targetPower:  200,
			controlPoint: controlPointBess,
		},
	}

	action := newTestController().prioritiseControlComponents(components)

	// Verify only the first component was used (greedy one)
	if action.activeComponentNames != "greedy" {
		t.Errorf("Expected only 'greedy' component, got %s", action.activeComponentNames)
	}
	if action.bessTargetPower != 100 {
		t.Errorf("Expected 100 power, got %f", action.bessTargetPower)
	}
}

func TestPrioritiseControlComponents_AllowMoreCharge(t *testing.T) {

	components := []controlComponent{
		{
			name:         "allow_more_charge",
			status:       componentStatusActiveAllowMoreCharge,
			targetPower:  -50, // Baseline
			controlPoint: controlPointBess,
		},
		{
			name:         "discharge_more", // discharge should be ignored
			status:       componentStatusActiveAllowMoreDischarge,
			targetPower:  200,
			controlPoint: controlPointBess,
		},
		{
			name:         "discharge_greedy", // discharge should be ignored
			status:       componentStatusActiveGreedy,
			targetPower:  50,
			controlPoint: controlPointBess,
		},
		{
			name:         "charge_more", // Should be allowed since it charges more
			status:       componentStatusActiveAllowMoreCharge,
			targetPower:  -100, // More charging (negative)
			controlPoint: controlPointBess,
		},
		{
			name:         "charge_less", // Should be ignored since it charges less
			status:       componentStatusActiveGreedy,
			targetPower:  -75, // Less charging (more positive)
			controlPoint: controlPointBess,
		},
	}

	action := newTestController().prioritiseControlComponents(components)

	// Verify the "charge_more" component was selected since it charges more
	if action.activeComponentNames != "allow_more_charge,charge_more" {
		t.Errorf("Expected 'allow_more_charge,charge_more', got %s", action.activeComponentNames)
	}
	if action.bessTargetPower != -100 {
		t.Errorf("Expected -100 power, got %f", action.bessTargetPower)
	}
}

func TestPrioritiseControlComponents_AllowMoreDischarge(t *testing.T) {

	components := []controlComponent{
		{
			name:         "inactive",
			status:       componentStatusInactive,
			targetPower:  0.0,
			controlPoint: controlPointBess,
		},
		{
			name:         "allow_more_discharge",
			status:       componentStatusActiveAllowMoreDischarge,
			targetPower:  50, // Baseline
			controlPoint: controlPointBess,
		},
		{
			name:         "charge_more", // charge should be ignored
			status:       componentStatusActiveAllowMoreCharge,
			targetPower:  -200,
			controlPoint: controlPointBess,
		},
		{
			name:         "charge_greedy", // charge should be ignored
			status:       componentStatusActiveGreedy,
			targetPower:  -50,
			controlPoint: controlPointBess,
		},
		{
			name:         "discharge_greedy", // Should be allowed since it discharges more
			status:       componentStatusActiveGreedy,
			targetPower:  100,
			controlPoint: controlPointBess,
		},
		{
			name:         "discharge_more", // Should be ignored since the previous component was greedy
			status:       componentStatusActiveGreedy,
			targetPower:  150,
			controlPoint: controlPointBess,
		},
	}

	action := newTestController().prioritiseControlComponents(components)

	// Verify the "charge_more" component was selected since it charges more
	if action.activeComponentNames != "allow_more_discharge,discharge_greedy" {
		t.Errorf("Expected 'allow_more_discharge,discharge_greedy', got %s", action.activeComponentNames)
	}
	if action.bessTargetPower != 100 {
		t.Errorf("Expected 100 power, got %f", action.bessTargetPower)
	}
}
