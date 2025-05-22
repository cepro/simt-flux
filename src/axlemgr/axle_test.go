package axlemgr

import (
	"sort"
	"testing"

	"github.com/cepro/besscontroller/axleclient"
	"github.com/cepro/besscontroller/telemetry"

	"github.com/stretchr/testify/assert"
)

func TestAxleMgr_getAxleReadings(t *testing.T) {

	// Test cases
	tests := []struct {
		name                string
		bessReading         *telemetry.BessReading
		bessMeterReading    *telemetry.MeterReading
		siteMeterReading    *telemetry.MeterReading
		axleAssetID         string
		bessNameplateEnergy float64
		expected            []axleclient.Reading
	}{
		{
			name:                "All readings are nil",
			bessReading:         nil,
			bessMeterReading:    nil,
			siteMeterReading:    nil,
			axleAssetID:         "asset-123",
			bessNameplateEnergy: 100.0,
			expected:            []axleclient.Reading{},
		},
		{
			name:             "Site meter reading with positive power",
			bessReading:      nil,
			bessMeterReading: nil,
			siteMeterReading: &telemetry.MeterReading{
				PowerTotalActive: pointerToFloat64(50.0),
			},
			axleAssetID:         "asset-123",
			bessNameplateEnergy: 100.0,
			expected: []axleclient.Reading{
				{
					AssetId: "asset-123",
					Value:   50.0,
					Label:   "boundary_import_kw",
				},
			},
		},
		{
			name:             "Site meter reading with negative power",
			bessReading:      nil,
			bessMeterReading: nil,
			siteMeterReading: &telemetry.MeterReading{
				PowerTotalActive: pointerToFloat64(-30.0),
			},
			axleAssetID:         "asset-123",
			bessNameplateEnergy: 100.0,
			expected: []axleclient.Reading{
				{
					AssetId: "asset-123",
					Value:   -30.0,
					Label:   "boundary_import_kw",
				},
			},
		},
		{
			name:        "BESS meter reading only",
			bessReading: nil,
			bessMeterReading: &telemetry.MeterReading{
				PowerTotalActive: pointerToFloat64(-70.0),
			},
			siteMeterReading:    nil,
			axleAssetID:         "asset-123",
			bessNameplateEnergy: 100.0,
			expected: []axleclient.Reading{
				{
					AssetId: "asset-123",
					Value:   70.0,
					Label:   "battery_inverter_import_kw",
				},
			},
		},
		{
			name: "BESS reading only",
			bessReading: &telemetry.BessReading{
				Soe: 75.0,
			},
			bessMeterReading:    nil,
			siteMeterReading:    nil,
			axleAssetID:         "asset-123",
			bessNameplateEnergy: 100.0,
			expected: []axleclient.Reading{
				{
					AssetId: "asset-123",
					Value:   75.0,
					Label:   "battery_state_of_charge_pct",
				},
			},
		},
		{
			name: "BESS reading 110% SoE", // this should be limited to 100% before being sent to Axle
			bessReading: &telemetry.BessReading{
				Soe: 110.0,
			},
			bessMeterReading:    nil,
			siteMeterReading:    nil,
			axleAssetID:         "asset-123",
			bessNameplateEnergy: 100.0,
			expected: []axleclient.Reading{
				{
					AssetId: "asset-123",
					Value:   100.0,
					Label:   "battery_state_of_charge_pct",
				},
			},
		},
		{
			name: "All readings present",
			bessReading: &telemetry.BessReading{
				Soe: 80.0,
			},
			bessMeterReading: &telemetry.MeterReading{
				PowerTotalActive: pointerToFloat64(70.0),
			},
			siteMeterReading: &telemetry.MeterReading{
				PowerTotalActive: pointerToFloat64(-20.0),
			},
			axleAssetID:         "asset-123",
			bessNameplateEnergy: 100.0,
			expected: []axleclient.Reading{
				{
					AssetId: "asset-123",
					Value:   -20.0,
					Label:   "boundary_import_kw",
				},
				{
					AssetId: "asset-123",
					Value:   -70,
					Label:   "battery_inverter_import_kw",
				},
				{
					AssetId: "asset-123",
					Value:   80,
					Label:   "battery_state_of_charge_pct",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			axleMgr := &AxleMgr{
				axleAssetID:         tc.axleAssetID,
				bessNameplateEnergy: tc.bessNameplateEnergy,
			}

			result := axleMgr.getAxleReadings(tc.bessReading, tc.bessMeterReading, tc.siteMeterReading)

			assertReadingsEqual(t, tc.expected, result)
		})
	}
}

// assertReadingsEqual compares two slices of axleclient.Reading and provides detailed output about differences.
// Doesn't compare start and end timestamps.
func assertReadingsEqual(t *testing.T, expected, actual []axleclient.Reading) {
	t.Helper() // Mark as a test helper function

	assert := assert.New(t)

	// Check length first
	if !assert.Equal(len(expected), len(actual),
		"Reading slices have different lengths. Expected: %d, Actual: %d", len(expected), len(actual)) {
		return
	}

	// Since readings might not be in the same order, we'll sort them first by Label
	expectedSorted := make([]axleclient.Reading, len(expected))
	actualSorted := make([]axleclient.Reading, len(actual))

	copy(expectedSorted, expected)
	copy(actualSorted, actual)

	sort.Slice(expectedSorted, func(i, j int) bool {
		return expectedSorted[i].Label < expectedSorted[j].Label
	})

	sort.Slice(actualSorted, func(i, j int) bool {
		return actualSorted[i].Label < actualSorted[j].Label
	})

	// Check each reading individually
	for i := range expectedSorted {
		expectedReading := expectedSorted[i]
		actualReading := actualSorted[i]

		// Compare AssetId
		assert.Equal(expectedReading.AssetId, actualReading.AssetId,
			"Reading[%d] AssetId mismatch. Expected: %s, Actual: %s",
			i, expectedReading.AssetId, actualReading.AssetId)

		// Compare Label
		assert.Equal(expectedReading.Label, actualReading.Label,
			"Reading[%d] Label mismatch. Expected: %s, Actual: %s",
			i, expectedReading.Label, actualReading.Label)

		// Compare Value with small epsilon for floating point comparison
		assert.InDelta(expectedReading.Value, actualReading.Value, 0.01,
			"Reading[%d] Value mismatch. Expected: %f, Actual: %f",
			i, expectedReading.Value, actualReading.Value)
	}
}

// Helper function to create float64 pointers
func pointerToFloat64(v float64) *float64 {
	return &v
}
