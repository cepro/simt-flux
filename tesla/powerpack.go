package tesla

import (
	"fmt"
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
// are received on the `p.Commands` channel.
func (p *PowerPack) Run(pollPeriod time.Duration) {

	pollTicker := time.NewTicker(pollPeriod)

	for {
		select {
		case command := <-p.Commands:
			// TODO: issue command
			fmt.Printf("Issue command to BESS: %v\n", command)
		case t := <-pollTicker.C:
			// TODO: read actual telemetry from the PowerPack
			p.Telemetry <- telemetry.BessReading{
				ID:     uuid.New(),
				Time:   t,
				BessID: p.id,
				Soe:    0.5,
			}
		}
	}
}
