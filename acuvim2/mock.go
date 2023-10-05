package acuvim2

import (
	"context"
	"time"

	"github.com/cepro/besscontroller/telemetry"
	"github.com/google/uuid"
)

type Acuvim2MeterMock struct {
	Telemetry chan telemetry.MeterReading
	id        uuid.UUID
}

func NewMock(id uuid.UUID, otherArgs ...interface{}) (*Acuvim2MeterMock, error) {
	return &Acuvim2MeterMock{
		Telemetry: make(chan telemetry.MeterReading),
		id:        id,
	}, nil
}

func (a *Acuvim2MeterMock) Run(ctx context.Context, period time.Duration) error {
	readingTicker := time.NewTicker(period)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case t := <-readingTicker.C:
			a.Telemetry <- telemetry.MeterReading{
				ReadingMeta: telemetry.ReadingMeta{
					ID:       uuid.New(),
					DeviceID: a.id,
					Time:     t,
				},
				Frequency:            50.0,
				VoltageLineAverage:   230.0,
				CurrentPhA:           10.1,
				CurrentPhB:           10.2,
				CurrentPhC:           10.3,
				CurrentPhAverage:     10.4,
				PowerPhAActive:       11.1,
				PowerPhBActive:       11.2,
				PowerPhCActive:       11.3,
				PowerTotalActive:     11.4,
				PowerTotalReactive:   12.5,
				PowerTotalApparent:   12.6,
				PowerFactorTotal:     1.0,
				EnergyImportedActive: 10000,
				EnergyExportedActive: 20000,
			}
		}
	}
}
