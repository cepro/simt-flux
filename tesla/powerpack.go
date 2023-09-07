package tesla

import (
	"context"
	"log/slog"
	"time"

	"github.com/cepro/besscontroller/telemetry"
	"github.com/google/uuid"
)

// PowerPack handles the communications with a Tesla Power Pack battery.
// Telemetry can be read from the `Telemetry` channel, and instructions for the battery can be
// sent into the `Commands` channel.
type PowerPack struct {
	Telemetry chan telemetry.BessReading
	Commands  chan telemetry.BessCommand
	id        uuid.UUID
	host      string
}

func NewPowerPack(id uuid.UUID, host string) (*PowerPack, error) {

	powerPack := &PowerPack{
		Telemetry: make(chan telemetry.BessReading),
		Commands:  make(chan telemetry.BessCommand),
		id:        id,
		host:      host,
	}

	// TODO: connect to actual power pack...

	return powerPack, nil
}

// Run loops forever, polling telemetry from the PowerPack every `pollPeriod`, and sending commands to the PowerPack every time they
// are received on the `p.Commands` channel. Exits when the context is cancelled.
func (p *PowerPack) Run(ctx context.Context, pollPeriod time.Duration) {

	pollTicker := time.NewTicker(pollPeriod)

	for {
		select {
		case <-ctx.Done():
			return
		case command := <-p.Commands:
			// TODO: issue command
			slog.Info("Issue command to BESS", "bess_command", command)
		case t := <-pollTicker.C:
			// TODO: read actual telemetry from the PowerPack
			p.Telemetry <- telemetry.BessReading{
				ReadingMeta: telemetry.ReadingMeta{
					ID:       uuid.New(),
					DeviceID: p.id,
					Time:     t,
				},
				Soe: 0.5,
			}
		}
	}
}
