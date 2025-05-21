package acuvim2

import (
	"context"
	"math/rand"
	"time"

	"github.com/cepro/besscontroller/telemetry"
	"github.com/google/uuid"
)

type Acuvim2MeterMock struct {
	readings chan<- telemetry.MeterReading
	id       uuid.UUID
}

func NewMock(readings chan<- telemetry.MeterReading, id uuid.UUID, otherArgs ...interface{}) (*Acuvim2MeterMock, error) {
	return &Acuvim2MeterMock{
		readings: readings,
		id:       id,
	}, nil
}

func (a *Acuvim2MeterMock) Run(ctx context.Context, period time.Duration) error {
	readingTicker := time.NewTicker(period)

	freq := 50.0
	powerTotalActive := 10.0
	energyImportedActive := 0.0
	energyExportedActive := 0.0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case t := <-readingTicker.C:
			a.readings <- telemetry.MeterReading{
				ReadingMeta: telemetry.ReadingMeta{
					ID:       uuid.New(),
					DeviceID: a.id,
					Time:     t,
				},
				Frequency:            &freq,
				PowerTotalActive:     &powerTotalActive,
				EnergyImportedActive: &energyImportedActive,
				EnergyExportedActive: &energyExportedActive,
			}
			energyImportedActive += rand.Float64()
			energyExportedActive += rand.Float64()
		}
	}
}
