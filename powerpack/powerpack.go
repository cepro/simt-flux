package powerpack

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/cepro/besscontroller/modbusaccess"
	"github.com/cepro/besscontroller/telemetry"
	"github.com/google/uuid"
	"github.com/grid-x/modbus"
)

const (
	MODBUS_TIMEOUT_SECS = uint16(10)
)

type PowerPack struct {
	Telemetry chan telemetry.BessReading
	Commands  chan telemetry.BessCommand

	host   string
	id     uuid.UUID
	client modbus.Client
	logger *slog.Logger

	heartbeatToggle        bool
	haveIssuedFirstCommand bool
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

	logger.Info("Connected, pulling PowerPack configuration....")

	p := &PowerPack{
		Telemetry:              make(chan telemetry.BessReading),
		Commands:               make(chan telemetry.BessCommand),
		id:                     id,
		host:                   host,
		client:                 client,
		logger:                 logger,
		heartbeatToggle:        false,
		haveIssuedFirstCommand: false,
	}

	// TODO: this failing will cause the whole app to fail - we might want something more resilient
	metrics, err := modbusaccess.PollBlock(p.client, p, configBlock)
	if err != nil {
		return nil, fmt.Errorf("poll config block: %w", err)
	}

	logger.Info(fmt.Sprintf("Retrieved PowerPack configuration: %+v", metrics))

	return p, nil
}

// Run loops forever polling telemetry from the meter every `period`. Exits when the context is cancelled.
func (p *PowerPack) Run(ctx context.Context, period time.Duration) error {

	readingTicker := time.NewTicker(period)

	// configure the heartbeat timeout for "direct real power commands" on the modbus connection
	modbusaccess.WriteRegister(p.client, directRealPowerCommandBlock.Registers["Timeout"], MODBUS_TIMEOUT_SECS)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case command := <-p.Commands:
			slog.Info("Issuing command to BESS", "bess_command", command)

			// The PowerPack expects the heartbeat to be toggled regularly
			modbusaccess.WriteRegister(p.client, directRealPowerCommandBlock.Registers["Heartbeat"], p.nextHeartbeat())

			// The PowerPack expects power in units of Watts
			modbusaccess.WriteRegister(p.client, directRealPowerCommandBlock.Registers["Power"], uint32(math.Round(command.TargetPower*1000)))

			// If this is the first power command we have issued, then set the "real power command mode" to "direct" (which means we will tell the PowerPack
			// direclty how much power to import/export)
			if !p.haveIssuedFirstCommand {
				modbusaccess.WriteRegister(p.client, realPowerCommandBlock.Registers["Mode"], uint16(1))
				p.haveIssuedFirstCommand = true
			}

		case t := <-readingTicker.C:

			metrics, err := modbusaccess.PollBlock(p.client, p, statusBlock)
			if err != nil {
				p.logger.Error("Failed to poll BESS", "error", err)
				continue // TODO: is this the right error handling
			}

			fmt.Printf("Got tesla metrics: %+v\n", metrics)

			p.Telemetry <- telemetry.BessReading{
				ReadingMeta: telemetry.ReadingMeta{
					ID:       uuid.New(),
					DeviceID: p.id,
					Time:     t,
				},
				TargetPower:             float64(metrics["BatteryTargetP"].(int32)) / 1000.0,
				Soe:                     float64(metrics["NominalEnergy"].(int32)) / 1000.0,
				AvailableInverterBlocks: metrics["AvailableBlocks"].(uint16),
				CommandSource:           metrics["CommandSource"].(uint16),
			}
		}
	}
}

// nextHeartbeat returns the heartbeat value to send to the PowerPack
func (p *PowerPack) nextHeartbeat() uint16 {
	p.heartbeatToggle = !p.heartbeatToggle
	if p.heartbeatToggle {
		return 0xAA55
	} else {
		return 0x55AA
	}
}
