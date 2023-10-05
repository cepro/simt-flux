package powerpack

import (
	"context"
	"log/slog"
	"time"

	"github.com/cepro/besscontroller/telemetry"
	"github.com/google/uuid"
)

type PowerPackMock struct {
	Telemetry chan telemetry.BessReading
	Commands  chan telemetry.BessCommand
	id        uuid.UUID
}

func NewMock(id uuid.UUID, otherArgs ...interface{}) (*PowerPackMock, error) {
	return &PowerPackMock{
		Telemetry: make(chan telemetry.BessReading),
		Commands:  make(chan telemetry.BessCommand),
		id:        id,
	}, nil
}

func (p *PowerPackMock) Run(ctx context.Context, period time.Duration) error {
	readingTicker := time.NewTicker(period)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case t := <-readingTicker.C:
			p.Telemetry <- telemetry.BessReading{
				ReadingMeta: telemetry.ReadingMeta{
					ID:       uuid.New(),
					DeviceID: p.id,
					Time:     t,
				},
				TargetPower: 30,
				Soe:         100,
			}
		case command := <-p.Commands:
			slog.Info("Issue command to BESS", "bess_command", command)
		}

	}
}
