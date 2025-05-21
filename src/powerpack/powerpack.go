package powerpack

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/cepro/besscontroller/modbus"
	"github.com/cepro/besscontroller/telemetry"
	"github.com/google/uuid"
)

const (
	MODBUS_TIMEOUT_SECS = uint16(10)
)

// PowerPack represents a Tesla battery (actually a PowerPack or a MegaPack)
type PowerPack struct {
	host            string
	id              uuid.UUID
	nameplateEnergy float64
	nameplatePower  float64

	teslaOptions TeslaOptions

	telemetry              chan telemetry.BessReading
	commands               chan telemetry.BessCommand
	client                 *modbus.Client
	heartbeatToggle        bool
	haveInitializedBess    bool
	haveIssuedFirstCommand bool
	logger                 *slog.Logger
}

// TeslaOptions defines parameters that are set internally on the PowerPack via modbus
type TeslaOptions struct {
	RampRateUp       float64 // sets the maximum ramp up rate at the inverters
	RampRateDown     float64 // sets the maximum ramp down rate at the inverters
	AlwaysActiveMode bool    // if true, then equipment will not enter power saving modes, meaning it is more responsive, but less efficient
}

func New(id uuid.UUID, host string, nameplateEnergy, nameplatePower float64, teslaOptions TeslaOptions) (*PowerPack, error) {

	logger := slog.Default().With("bess_id", id, "host", host)

	client, err := modbus.NewClient(host)
	if err != nil {
		return nil, fmt.Errorf("create modbus client: %w", err)
	}

	p := &PowerPack{
		host:                   host,
		id:                     id,
		nameplateEnergy:        nameplateEnergy,
		nameplatePower:         nameplatePower,
		teslaOptions:           teslaOptions,
		telemetry:              make(chan telemetry.BessReading, 1),
		commands:               make(chan telemetry.BessCommand, 1),
		client:                 client,
		heartbeatToggle:        false,
		haveInitializedBess:    false,
		haveIssuedFirstCommand: false,
		logger:                 logger,
	}

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

		case t := <-readingTicker.C:

			metricVals, err := p.client.PollBlock(nil, statusBlock)
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
				TargetPower:             float64(metricVals["BatteryTargetP"].(int32)) / 1000.0,
				Soe:                     float64(metricVals["NominalEnergy"].(int32)) / 1000.0,
				AvailableInverterBlocks: metricVals["AvailableBlocks"].(uint16),
				CommandSource:           metricVals["CommandSource"].(uint16),
			}
		}
	}
}

// initializeBessIfRequired runs through the intial configuration of the PowerPack, if it hasn't already been done.
func (p *PowerPack) initializeBessIfRequired() error {

	if p.haveInitializedBess {
		return nil
	}

	err := p.client.WriteMetric(realPowerRampParametersBlock.Metrics["RampUp"], uint32(p.teslaOptions.RampRateUp*1000)) // kW/s to W/s
	if err != nil {
		return fmt.Errorf("set ramp up rate: %w", err)
	}

	err = p.client.WriteMetric(realPowerRampParametersBlock.Metrics["RampDown"], uint32(p.teslaOptions.RampRateDown*1000)) // kW/s to W/s
	if err != nil {
		return fmt.Errorf("set ramp down rate: %w", err)
	}

	err = p.client.WriteMetric(realPowerCommandBlock.Metrics["AlwaysActive"], boolToUint16(p.teslaOptions.AlwaysActiveMode))
	if err != nil {
		return fmt.Errorf("set always active mode: %w", err)
	}

	p.logger.Info(fmt.Sprintf("Applied powerpack tesla options: %+v", p.teslaOptions))

	p.haveInitializedBess = true

	p.logConfigParameters()

	return nil
}

// logConfigParameters reads the PowerPacks configuration parameters over modbus and logs them, errors are swallowed.
func (p *PowerPack) logConfigParameters() {
	metrics, err := p.client.PollBlock(nil, configBlock)
	if err != nil {
		p.logger.Error("Failed to retrieve PowerPack configuration", "error", err)
		return
	}
	p.logger.Info(fmt.Sprintf("Retrieved PowerPack configuration: %+v", metrics))

	metrics, err = p.client.PollBlock(nil, realPowerRampParametersBlock)
	if err != nil {
		p.logger.Error("Failed to retrieve PowerPack ramp configuration", "error", err)
		return
	}
	p.logger.Info(fmt.Sprintf("Retrieved PowerPack ramp configuration: %+v", metrics))

	metrics, err = p.client.PollBlock(nil, realPowerCommandBlock)
	if err != nil {
		p.logger.Error("Failed to retrieve PowerPack real power command configuration", "error", err)
		return
	}
	p.logger.Info(fmt.Sprintf("Retrieved PowerPack real power command configuration: %+v", metrics))
}

// issueCommand sends the given command to the PowerPack and manages the associated modbus registers like heartbeat, timeout and real power mode.
func (p *PowerPack) issueCommand(command telemetry.BessCommand) error {

	err := p.initializeBessIfRequired()
	if err != nil {
		return fmt.Errorf("initialize bess: %w", err)
	}

	// The PowerPack expects the heartbeat to be toggled regularly
	err = p.client.WriteMetric(directRealPowerCommandBlock.Metrics["Heartbeat"], p.nextHeartbeat())
	if err != nil {
		return fmt.Errorf("write heartbeat: %w", err)
	}

	// The PowerPack expects power in units of Watts
	p.client.WriteMetric(directRealPowerCommandBlock.Metrics["Power"], uint32(math.Round(command.TargetPower*1000)))
	if err != nil {
		return fmt.Errorf("write real power: %w", err)
	}

	// If this is the first power command we have issued, then set the "real power command mode" to "direct" (which means we will tell the PowerPack
	// direclty how much power to import/export). The Tesla manual reccomends setting this *after* the first power command, hence this is not sent
	// in the `initializeBessIfRequired` function.
	if !p.haveIssuedFirstCommand {
		// configure the heartbeat timeout for "direct real power commands" on the modbus connection
		err = p.client.WriteMetric(directRealPowerCommandBlock.Metrics["Timeout"], MODBUS_TIMEOUT_SECS)
		if err != nil {
			return fmt.Errorf("write timeout: %w", err)
		}
		err = p.client.WriteMetric(realPowerCommandBlock.Metrics["Mode"], uint16(1))
		if err != nil {
			return fmt.Errorf("write real power mode: %w", err)
		}
		p.haveIssuedFirstCommand = true
	}

	return nil
}

func (p *PowerPack) ID() uuid.UUID {
	return p.id
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

// boolToUint16 converts a boolean value to an integer for transmission over modbus
func boolToUint16(b bool) uint16 {
	if b {
		return 1
	} else {
		return 0
	}
}
