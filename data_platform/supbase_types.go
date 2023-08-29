package dataplatform

import (
	"time"

	"github.com/cepro/besscontroller/repository"
	"github.com/google/uuid"
)

const (
	SUPABASE_BESS_READING_TABLE_NAME  = "bess_readings"
	SUPABASE_METER_READING_TABLE_NAME = "meter_readings"
)

// supabaseBessReading holds the json encoding schema for a BESS reading in supabase.
type supabaseBessReading struct {
	ID          uuid.UUID `json:"id"`
	Time        time.Time `json:"time"`
	BessID      uuid.UUID `json:"bess_id"`
	Soe         float64   `json:"soe"`
	TargetPower float64   `json:"target_power"`
}

// supabaseMeterReading holds the json encoding schema for a meter reading in supabase.
type supabaseMeterReading struct {
	ID         uuid.UUID `json:"id"`
	Time       time.Time `json:"time"`
	MeterID    uuid.UUID `json:"meter_id"`
	Frequency  float64   `json:"frequency"`
	TotalPower float64   `json:"total_power"`
}

// convertReadingsForSupabase returns the equivilent "supbase type" for the given readings (which include supabase json tags) and the
// associated supabase table name.
func convertReadingsForSupabase(readings interface{}) (interface{}, string) {
	switch readingsTyped := readings.(type) {

	case []repository.StoredBessReading:
		var supabaseReadings []supabaseBessReading
		for _, reading := range readingsTyped {
			supabaseReadings = append(supabaseReadings, supabaseBessReading(reading.BessReading))
		}
		return supabaseReadings, SUPABASE_BESS_READING_TABLE_NAME

	case []repository.StoredMeterReading:
		var supabaseReadings []supabaseMeterReading
		for _, reading := range readingsTyped {
			supabaseReadings = append(supabaseReadings, supabaseMeterReading(reading.MeterReading))
		}
		return supabaseReadings, SUPABASE_METER_READING_TABLE_NAME

	default:
		return nil, ""
	}
}
