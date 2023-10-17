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
	host            string
	id              uuid.UUID
	nameplateEnergy float64
	nameplatePower  float64

	telemetry              chan telemetry.BessReading
	commands               chan telemetry.BessCommand
	client                 modbus.Client
	heartbeatToggle        bool
	haveIssuedFirstCommand bool
	logger                 *slog.Logger
}

func New(id uuid.UUID, host string, nameplateEnergy, nameplatePower float64) (*PowerPack, error) {

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
		host:                   host,
		id:                     id,
		nameplateEnergy:        nameplateEnergy,
		nameplatePower:         nameplatePower,
		telemetry:              make(chan telemetry.BessReading, 1),
		commands:               make(chan telemetry.BessCommand, 1),
		client:                 client,
		heartbeatToggle:        false,
		haveIssuedFirstCommand: false,
		logger:                 logger,
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

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case command := <-p.commands:
			err := p.issueCommand(command)
			if err != nil {
				p.logger.Error("Failed to issue command to bess", "bess_command", command, "error", err)
				continue
			}
			p.logger.Info("Issued command to BESS", "bess_command", command)

		case t := <-readingTicker.C:

			metrics, err := modbusaccess.PollBlock(p.client, p, statusBlock)
			if err != nil {
				p.logger.Error("Failed to poll BESS", "error", err)
				continue // TODO: is this the right error handling
			}

			p.telemetry <- telemetry.BessReading{
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

// issueCommand sends the given command to the PowerPack and manages the associated modbus registers like heartbeat, timeout and real power mode.
func (p *PowerPack) issueCommand(command telemetry.BessCommand) error {
	// The PowerPack expects the heartbeat to be toggled regularly
	err := modbusaccess.WriteRegister(p.client, directRealPowerCommandBlock.Registers["Heartbeat"], p.nextHeartbeat())
	if err != nil {
		return fmt.Errorf("write heartbeat: %w", err)
	}

	// The PowerPack expects power in units of Watts
	modbusaccess.WriteRegister(p.client, directRealPowerCommandBlock.Registers["Power"], uint32(math.Round(command.TargetPower*1000)))
	if err != nil {
		return fmt.Errorf("write real power: %w", err)
	}

	// If this is the first power command we have issued, then set the "real power command mode" to "direct" (which means we will tell the PowerPack
	// direclty how much power to import/export)
	if !p.haveIssuedFirstCommand {
		// configure the heartbeat timeout for "direct real power commands" on the modbus connection
		err = modbusaccess.WriteRegister(p.client, directRealPowerCommandBlock.Registers["Timeout"], MODBUS_TIMEOUT_SECS)
		if err != nil {
			return fmt.Errorf("write timeout: %w", err)
		}
		err = modbusaccess.WriteRegister(p.client, realPowerCommandBlock.Registers["Mode"], uint16(1))
		if err != nil {
			return fmt.Errorf("write real power mode: %w", err)
		}
		p.haveIssuedFirstCommand = true
	}

	return nil
}

func (p *PowerPack) NameplateEnergy() float64 {
	return p.nameplateEnergy
}

func (p *PowerPack) NameplatePower() float64 {
	return p.nameplatePower
}

func (p *PowerPack) Commands() chan<- telemetry.BessCommand {
	return p.commands
}

func (p *PowerPack) Telemetry() <-chan telemetry.BessReading {
	return p.telemetry
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
