package powerpack

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/cepro/besscontroller/modbusaccess"
	"github.com/cepro/besscontroller/telemetry"
	"github.com/google/uuid"
	"github.com/grid-x/modbus"
)

type PowerPack struct {
	Telemetry chan telemetry.BessReading
	Commands  chan telemetry.BessCommand

	host   string
	id     uuid.UUID
	client modbus.Client
	logger *slog.Logger
}

func New(id uuid.UUID, host string) (*PowerPack, error) {

	logger := slog.Default().With("bess_id", id, "host", host)

	handler := modbus.NewTCPClientHandler(host)
	handler.Timeout = 10 * time.Second
	handler.SlaveID = 0x01

	logger.Info("Connecting to Tesla PowerPack...")

	err := handler.Connect()
	if err != nil {
		return nil, err
	}
	defer handler.Close()

	client := modbus.NewClient(handler)

	logger.Info("Connected")

	return &PowerPack{
		Telemetry: make(chan telemetry.BessReading),
		Commands:  make(chan telemetry.BessCommand),
		id:        id,
		host:      host,
		client:    client,
		logger:    logger,
	}, nil
}

// Run loops forever polling telemetry from the meter every `period`. Exits when the context is cancelled.
func (p *PowerPack) Run(ctx context.Context, period time.Duration) error {

	readingTicker := time.NewTicker(period)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case command := <-p.Commands:
			// TODO: issue command
			slog.Info("Issue command to BESS", "bess_command", command)

		case <-readingTicker.C:

			metrics, err := modbusaccess.PollBlocks(p.client, p, blocks)
			if err != nil {
				p.logger.Error("Failed to poll BESS", "error", err)
				continue // TODO: is this the right error handling
			}

			fmt.Printf("Got tesla metrics: %+v\n", metrics)

			// p.Telemetry <- telemetry.BessReading{
			// 	ReadingMeta: telemetry.ReadingMeta{
			// 		ID:       uuid.New(),
			// 		DeviceID: p.id,
			// 		Time:     t,
			// 	},
			// 	Soe: 0.5,
			// }
		}
	}
}
