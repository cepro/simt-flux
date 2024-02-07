package supabase

import (
	"fmt"
	"time"

	"github.com/cepro/besscontroller/telemetry"
	"github.com/google/uuid"
)

const (
	SUPABASE_BESS_READING_TABLE_NAME  = "mg_bess_readings"
	SUPABASE_METER_READING_TABLE_NAME = "mg_meter_readings"
)

type SupabaseReadingMeta struct {
	ID       uuid.UUID `json:"id"`
	DeviceID uuid.UUID `json:"device_id"`
	Time     time.Time `json:"time"`
}

// supabaseBessReading holds the json encoding schema for a BESS reading in supabase.
type supabaseBessReading struct {
	SupabaseReadingMeta
	Soe         float64 `json:"soe"`
	TargetPower float64 `json:"target_power"`
}

// supabaseMeterReading holds the json encoding schema for a meter reading in supabase.
type supabaseMeterReading struct {
	SupabaseReadingMeta

	Frequency               *float64 `json:"frequency"`
	VoltageLineAverage      *float64 `json:"voltage_line_average"`
	CurrentPhA              *float64 `json:"current_phase_a"`
	CurrentPhB              *float64 `json:"current_phase_b"`
	CurrentPhC              *float64 `json:"current_phase_c"`
	CurrentPhAverage        *float64 `json:"current_phase_average"`
	PowerPhAActive          *float64 `json:"power_phase_a_active"`
	PowerPhBActive          *float64 `json:"power_phase_b_active"`
	PowerPhCActive          *float64 `json:"power_phase_c_active"`
	PowerTotalActive        *float64 `json:"power_total_active"`
	PowerTotalReactive      *float64 `json:"power_total_reactive"`
	PowerTotalApparent      *float64 `json:"power_total_apparent"`
	PowerFactorTotal        *float64 `json:"power_factor_total"`
	EnergyImportedActive    *float64 `json:"energy_imported_active"`
	EnergyExportedActive    *float64 `json:"energy_exported_active"`
	EnergyImportedPhAActive *float64 `json:"energy_imported_phase_a_active"`
	EnergyExportedPhAActive *float64 `json:"energy_exported_phase_a_active"`
	EnergyImportedPhBActive *float64 `json:"energy_imported_phase_b_active"`
	EnergyExportedPhBActive *float64 `json:"energy_exported_phase_b_active"`
	EnergyImportedPhCActive *float64 `json:"energy_imported_phase_c_active"`
	EnergyExportedPhCActive *float64 `json:"energy_exported_phase_c_active"`
}

// convertReadingsForSupabase returns the equivilent "supbase type" for the given readings (which include supabase json tags) and the
// associated supabase table name.
func convertReadingsForSupabase(readings interface{}) (interface{}, string) {
	switch readingsTyped := readings.(type) {

	case []telemetry.BessReading:
		supabaseReadings := make([]supabaseBessReading, 0, len(readingsTyped))
		for _, reading := range readingsTyped {
			supabaseReadings = append(supabaseReadings, supabaseBessReading{
				SupabaseReadingMeta: SupabaseReadingMeta(reading.ReadingMeta),
				Soe:                 reading.Soe,
				TargetPower:         reading.TargetPower,
			})
		}
		return supabaseReadings, SUPABASE_BESS_READING_TABLE_NAME

	case []telemetry.MeterReading:
		supabaseReadings := make([]supabaseMeterReading, 0, len(readingsTyped))
		for _, reading := range readingsTyped {
			supabaseReadings = append(supabaseReadings, supabaseMeterReading{
				SupabaseReadingMeta:     SupabaseReadingMeta(reading.ReadingMeta),
				Frequency:               reading.Frequency,
				VoltageLineAverage:      reading.VoltageLineAverage,
				CurrentPhA:              reading.CurrentPhA,
				CurrentPhB:              reading.CurrentPhB,
				CurrentPhC:              reading.CurrentPhC,
				CurrentPhAverage:        reading.CurrentPhAverage,
				PowerPhAActive:          reading.PowerPhAActive,
				PowerPhBActive:          reading.PowerPhBActive,
				PowerPhCActive:          reading.PowerPhCActive,
				PowerTotalActive:        reading.PowerTotalActive,
				PowerTotalReactive:      reading.PowerTotalReactive,
				PowerTotalApparent:      reading.PowerTotalApparent,
				PowerFactorTotal:        reading.PowerFactorTotal,
				EnergyImportedActive:    reading.EnergyImportedActive,
				EnergyExportedActive:    reading.EnergyExportedActive,
				EnergyImportedPhAActive: reading.EnergyImportedPhAActive,
				EnergyExportedPhAActive: reading.EnergyExportedPhAActive,
				EnergyImportedPhBActive: reading.EnergyImportedPhBActive,
				EnergyExportedPhBActive: reading.EnergyExportedPhBActive,
				EnergyImportedPhCActive: reading.EnergyImportedPhCActive,
				EnergyExportedPhCActive: reading.EnergyExportedPhCActive,
			})
		}
		return supabaseReadings, SUPABASE_METER_READING_TABLE_NAME

	default:
		panic(fmt.Sprintf("Unknown readings type: '%T'", readings))
	}
}
