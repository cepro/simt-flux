package powerpack

import (
	"context"
	"log/slog"
	"time"

	"github.com/cepro/besscontroller/telemetry"
	"github.com/google/uuid"
)

type PowerPackMock struct {
	id              uuid.UUID
	telemetry       chan telemetry.BessReading
	commands        chan telemetry.BessCommand
	nameplateEnergy float64
	nameplatePower  float64
}

func NewMock(id uuid.UUID, nameplateEnergy, nameplatePower float64) (*PowerPackMock, error) {
	return &PowerPackMock{
		id:              id,
		telemetry:       make(chan telemetry.BessReading, 1),
		commands:        make(chan telemetry.BessCommand, 1),
		nameplateEnergy: nameplateEnergy,
		nameplatePower:  nameplatePower,
	}, nil
}

func (p *PowerPackMock) Run(ctx context.Context, period time.Duration) error {
	readingTicker := time.NewTicker(period)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case t := <-readingTicker.C:
			p.telemetry <- telemetry.BessReading{
				ReadingMeta: telemetry.ReadingMeta{
					ID:       uuid.New(),
					DeviceID: p.id,
					Time:     t,
				},
				TargetPower: 30,
				Soe:         100,
			}
		case command := <-p.commands:
			slog.Info("Issue command to BESS", "bess_command", command)
		}

	}
}

func (p *PowerPackMock) ID() uuid.UUID {
	return p.id
}

func (p *PowerPackMock) NameplateEnergy() float64 {
	return p.nameplateEnergy
}

func (p *PowerPackMock) NameplatePower() float64 {
	return p.nameplatePower
}

func (p *PowerPackMock) Commands() chan<- telemetry.BessCommand {
	return p.commands
}

func (p *PowerPackMock) Telemetry() <-chan telemetry.BessReading {
	return p.telemetry
}
