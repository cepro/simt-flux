package acuvim2

import (
	"errors"
	"time"

	"github.com/cepro/besscontroller/telemetry"
	"github.com/google/uuid"
)

// EmulatedAcuvim2Meter looks like an Acuvim2Meter but produces fake data.
type EmulatedAcuvim2Meter struct {
	Telemetry chan telemetry.MeterReading
	id        uuid.UUID
}

func NewEmulated(id uuid.UUID) (*EmulatedAcuvim2Meter, error) {

	meter := &EmulatedAcuvim2Meter{
		Telemetry: make(chan telemetry.MeterReading),
		id:        id,
	}
	return meter, nil
}

func (a *EmulatedAcuvim2Meter) Run(period time.Duration) error {

	readingTicker := time.NewTicker(period)

	for t := range readingTicker.C {

		frequency := 50.0
		totalPower := 10e3

		a.Telemetry <- telemetry.MeterReading{
			ID:         uuid.New(),
			Time:       t,
			MeterID:    a.id,
			Frequency:  frequency,
			TotalPower: totalPower,
		}
	}

	return errors.New("Channel closed")
}
