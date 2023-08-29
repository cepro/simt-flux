package dataplatform

import (
	"time"

	"github.com/cepro/besscontroller/repository"
	"github.com/google/uuid"
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

func convertMeterReadings(readings []repository.StoredMeterReading) []supabaseMeterReading {
	var supabaseReadings []supabaseMeterReading
	for _, reading := range readings {
		supabaseReadings = append(supabaseReadings, supabaseMeterReading(reading.MeterReading))
	}
	return supabaseReadings
}

func convertBessReadings(readings []repository.StoredBessReading) []supabaseBessReading {
	var supabaseReadings []supabaseBessReading
	for _, reading := range readings {
		supabaseReadings = append(supabaseReadings, supabaseBessReading(reading.BessReading))
	}
	return supabaseReadings
}
