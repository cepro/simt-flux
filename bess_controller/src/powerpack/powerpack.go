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

// PowerPack represents a Tesla battery (this actually supports both PowerPacks and MegaPacks as they use a similar modbus API)
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

	// Idle-based OFF mode state tracking
	currentMode   uint16     // Current Tesla mode read from battery (ground truth)
	idleStartTime *time.Time // When power=0 started, nil if power≠0

	// Shutdown state tracking for proper battery OFF/ON sequences
	shutdownState       uint16     // Current shutdown register value: 0=active, 1=shutdown
	safetyPeriodEnd     *time.Time // When 60s safety period expires (nil when not in safety period)
	pendingModeWrite    bool       // Flag: need to write Mode=1 after safety period expires
	controllerStartTime time.Time  // When controller started (used to prevent turn ON during first 60s)

	logger *slog.Logger
}

// TeslaOptions defines parameters that are set internally on the PowerPack via modbus
type TeslaOptions struct {
	RampRateUp       float64 // sets the maximum ramp up rate at the inverters
	RampRateDown     float64 // sets the maximum ramp down rate at the inverters
	AlwaysActiveMode bool    // if true, then equipment will not enter power saving modes, meaning it is more responsive, but less efficient

	// Idle-based OFF mode configuration
	OffIdleEnabled       bool    // Enable automatic turn OFF after idle period
	OffIdleThresholdMins float64 // Minutes of power=0 before turning OFF (e.g., 5.0)
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

		// Initialize OFF mode tracking
		currentMode:   0,   // Will be read from battery
		idleStartTime: nil, // No idle period yet

		// Initialize shutdown state tracking
		shutdownState:       0,            // Will be read from battery (assume active initially)
		safetyPeriodEnd:     nil,          // Not in safety period
		pendingModeWrite:    false,        // No pending operations
		controllerStartTime: time.Now(),   // Track startup time for 60s safety delay

		logger: logger,
	}

	return p, nil
}

// Run loops forever polling telemetry from the Tesla battery every `period`. Exits when the context is cancelled.
func (p *PowerPack) Run(ctx context.Context, period time.Duration) error {

	readingTicker := time.NewTicker(period)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case command := <-p.commands: // if we receive a command then send it to the battery
			err := p.issueCommand(command)
			if err != nil {
				p.logger.Error("Failed to issue command to bess", "bess_command", command, "error", err)
				continue
			}

		case t := <-readingTicker.C: // poll telemetry regularly

			metricVals, err := p.client.PollBlock(nil, statusBlock)
			if err != nil {
				p.logger.Error("Failed to poll BESS status", "error", err)
				continue // try again next time
			}

			// Read mode block to get current Tesla mode (ground truth for OFF mode logic)
			modeVals, err := p.client.PollBlock(nil, realPowerCommandBlock)
			if err != nil {
				p.logger.Error("Failed to poll BESS mode", "error", err)
				// Don't fail completely - use cached mode value
				modeVals = map[string]interface{}{"Mode": p.currentMode}
			}

			// Update cached mode with actual value from Tesla
			readMode := modeVals["Mode"].(uint16)
			if readMode != p.currentMode {
				p.logger.Info("Battery mode changed", "old_mode", p.currentMode, "new_mode", readMode)
			}
			p.currentMode = readMode

			// Read shutdown state block to get current shutdown register value
			shutdownVals, err := p.client.PollBlock(nil, shutdownCommandBlock)
			if err != nil {
				p.logger.Error("Failed to poll BESS shutdown state", "error", err)
				// Don't fail completely - use cached shutdown value
				shutdownVals = map[string]interface{}{"Shutdown": p.shutdownState}
			}

			// Update cached shutdown state with actual value from Tesla
			readShutdown := shutdownVals["Shutdown"].(uint16)
			shutdownInterpretation := map[uint16]string{0: "active", 1: "shutdown"}

			if readShutdown != p.shutdownState {
				p.logger.Info("Battery shutdown state read",
					"old_shutdown", p.shutdownState,
					"new_shutdown", readShutdown,
					"interpretation", shutdownInterpretation[readShutdown],
					"first_read_on_startup", p.shutdownState == 0 && !p.haveIssuedFirstCommand)
			}
			p.shutdownState = readShutdown

			// Check if safety period has expired and complete any pending operations
			err = p.checkSafetyPeriod()
			if err != nil {
				p.logger.Error("Failed to complete safety period operations", "error", err)
				// Don't fail completely - will retry next poll
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
				RealPowerMode:           readMode, // Include actual mode from Tesla
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

// issueCommand sends commands to the PowerPack and optionally manages idle-based OFF mode.
//
// When idle OFF is enabled, the battery automatically turns OFF after being idle (power=0)
// for a configurable duration to reduce standby power consumption (~200W).
//
// Proper shutdown/startup sequences using Shutdown register (1500):
//   TURN OFF: Mode=0 → wait 5s → Shutdown=1 → 60s safety period → complete
//   TURN ON:  Shutdown=0 → 60s safety period → Mode=1 + Timeout → complete
//
// The idle OFF feature is disabled by default and must be explicitly enabled in config:
//   teslaOptions:
//     offIdleEnabled: true          # Default: false
//     offIdleThresholdMins: 5.0     # Minutes of idle before turning OFF
func (p *PowerPack) issueCommand(command telemetry.BessCommand) error {

	err := p.initializeBessIfRequired()
	if err != nil {
		return fmt.Errorf("initialize bess: %w", err)
	}

	// Track idle time when feature is enabled or for debugging
	now := time.Now()
	if p.teslaOptions.OffIdleEnabled {
		if command.TargetPower == 0.0 {
			if p.idleStartTime == nil {
				p.idleStartTime = &now
				p.logger.Debug("Battery entered idle state (power=0)")
			}
		} else {
			if p.idleStartTime != nil {
				idleDuration := now.Sub(*p.idleStartTime)
				p.logger.Debug("Battery exited idle state",
					"idle_duration_secs", idleDuration.Seconds())
			}
			p.idleStartTime = nil
		}
	}

	// Calculate idle duration
	idleDuration := time.Duration(0)
	if p.idleStartTime != nil {
		idleDuration = now.Sub(*p.idleStartTime)
	}

	// Determine if we should turn OFF due to idle timeout
	shouldTurnOff := false
	if p.teslaOptions.OffIdleEnabled &&
		command.TargetPower == 0.0 &&
		idleDuration >= time.Duration(p.teslaOptions.OffIdleThresholdMins*float64(time.Minute)) {
		shouldTurnOff = true
	}

	// REJECT COMMANDS DURING SAFETY PERIOD
	// Battery is in 60s safety period after shutdown/startup - don't interfere
	if p.safetyPeriodEnd != nil && now.Before(*p.safetyPeriodEnd) {
		remaining := time.Until(*p.safetyPeriodEnd)
		p.logger.Debug("Battery in safety period, skipping command",
			"remaining_secs", remaining.Seconds(),
			"pending_mode_write", p.pendingModeWrite)
		return nil
	}

	// TURN OFF: Battery is ON and idle timeout reached
	if p.currentMode == 1 && p.shutdownState == 0 && shouldTurnOff {
		p.logger.Info("Turning battery OFF due to idle timeout",
			"idle_duration_mins", idleDuration.Minutes(),
			"threshold_mins", p.teslaOptions.OffIdleThresholdMins)

		err = p.turnBatteryOff()
		if err != nil {
			return fmt.Errorf("turn battery OFF: %w", err)
		}

		p.haveIssuedFirstCommand = true
		// Don't send heartbeat or power during shutdown transition
		return nil
	}

	// TURN ON: Battery is OFF/shutdown and power is needed
	// Only turn ON if: first startup OR power demand exists
	shouldTurnOn := !p.haveIssuedFirstCommand || command.TargetPower != 0.0

	// PREVENT TURN ON DURING FIRST 60s AFTER CONTROLLER STARTUP
	// This protects against violating safety period if controller restarted during shutdown
	// ONLY applies when Shutdown=1 (unknown when it was written, must respect safety period)
	if p.shutdownState == 1 && shouldTurnOn {
		timeSinceStartup := now.Sub(p.controllerStartTime)
		if timeSinceStartup < 60*time.Second {
			remaining := 60*time.Second - timeSinceStartup
			p.logger.Info("Battery shutdown state=1, waiting 60s before turn ON",
				"time_since_startup_secs", timeSinceStartup.Seconds(),
				"remaining_secs", remaining.Seconds(),
				"reason", "Unknown when Shutdown=1 was written, respecting safety period")
			// Don't return error - just defer the operation, will retry on next command
			return nil
		}
	}
	// If Shutdown=0, proceed immediately - battery is safe to operate

	if (p.currentMode == 0 || p.shutdownState == 1) && shouldTurnOn {
		logMsg := "Initializing battery from shutdown/OFF state"
		if p.haveIssuedFirstCommand {
			logMsg = "Turning battery ON (power demand)"
		}
		p.logger.Info(logMsg,
			"current_mode", p.currentMode,
			"shutdown_state", p.shutdownState,
			"target_power", command.TargetPower)

		if p.teslaOptions.OffIdleEnabled {
			// Use proper shutdown register sequences (WLCE)
			err = p.turnBatteryOn()
			if err != nil {
				return fmt.Errorf("turn battery ON: %w", err)
			}
			p.haveIssuedFirstCommand = true
			p.idleStartTime = nil
			// Don't send heartbeat or power during startup transition
			return nil
		} else {
			// Simple Mode=1 write without shutdown register (Hazelmead)
			err = p.client.WriteMetric(realPowerCommandBlock.Metrics["Mode"], uint16(1))
			if err != nil {
				return fmt.Errorf("turn ON - write mode: %w", err)
			}
			err = p.client.WriteMetric(directRealPowerCommandBlock.Metrics["Timeout"], MODBUS_TIMEOUT_SECS)
			if err != nil {
				return fmt.Errorf("turn ON - write timeout: %w", err)
			}
			p.haveIssuedFirstCommand = true
			p.idleStartTime = nil
			// Fall through to send heartbeat + power
		}
	}

	// KEEP BATTERY OFF: Don't send commands if battery should stay OFF
	if (p.currentMode == 0 || p.shutdownState == 1) && !shouldTurnOn {
		p.logger.Debug("Battery remains OFF (no power demand)",
			"current_mode", p.currentMode,
			"shutdown_state", p.shutdownState,
			"target_power", command.TargetPower)
		return nil
	}

	// Always send heartbeat and power for normal operation
	// Write heartbeat toggle
	err = p.client.WriteMetric(directRealPowerCommandBlock.Metrics["Heartbeat"], p.nextHeartbeat())
	if err != nil {
		return fmt.Errorf("write heartbeat: %w", err)
	}

	// Write target power
	err = p.client.WriteMetric(directRealPowerCommandBlock.Metrics["Power"], uint32(math.Round(command.TargetPower*1000)))
	if err != nil {
		return fmt.Errorf("write real power: %w", err)
	}

	return nil
}

// turnBatteryOff initiates the proper Tesla shutdown sequence:
// Step 1: Write Mode=0
// Step 2: Wait 5 seconds (short blocking delay - acceptable telemetry gap)
// Step 3: Write Shutdown=1
// Step 4: Start 60s safety period timer (non-blocking)
//
// Returns error if already in safety period or if ModbusTCP write fails.
func (p *PowerPack) turnBatteryOff() error {
	// Prevent operations during safety period
	if p.safetyPeriodEnd != nil {
		remaining := time.Until(*p.safetyPeriodEnd)
		return fmt.Errorf("battery in safety period, cannot turn OFF (remaining: %.1fs)", remaining.Seconds())
	}

	p.logger.Info("Starting battery shutdown sequence",
		"current_mode", p.currentMode,
		"current_shutdown", p.shutdownState)

	// Step 1: Write Mode=0 (OFF)
	err := p.client.WriteMetric(realPowerCommandBlock.Metrics["Mode"], uint16(0))
	if err != nil {
		return fmt.Errorf("shutdown - write Mode=0: %w", err)
	}

	p.logger.Info("Wrote Mode=0, waiting 5s before Shutdown=1")

	// Step 2: Short blocking sleep (5s telemetry gap is acceptable during shutdown)
	time.Sleep(5 * time.Second)

	// Step 3: Write Shutdown=1
	err = p.client.WriteMetric(shutdownCommandBlock.Metrics["Shutdown"], uint16(1))
	if err != nil {
		return fmt.Errorf("shutdown - write Shutdown=1: %w", err)
	}

	// Update cached shutdown state immediately after successful write
	p.shutdownState = 1

	// Step 4: Start 60s safety period (non-blocking)
	safetyEnd := time.Now().Add(60 * time.Second)
	p.safetyPeriodEnd = &safetyEnd
	p.pendingModeWrite = false // No pending operations for shutdown

	p.logger.Info("Shutdown sequence initiated, 60s safety period started",
		"safety_period_end", safetyEnd.Format(time.RFC3339))

	return nil
}

// turnBatteryOn initiates the proper Tesla startup sequence:
// Step 1: Write Shutdown=0
// Step 2: Start 60s safety period timer with pending Mode=1 write
//
// The Mode=1 write will be completed by checkSafetyPeriod() after 60s expires.
//
// Returns error if already in safety period or if ModbusTCP write fails.
func (p *PowerPack) turnBatteryOn() error {
	// Prevent operations during safety period
	if p.safetyPeriodEnd != nil {
		remaining := time.Until(*p.safetyPeriodEnd)
		return fmt.Errorf("battery in safety period, cannot turn ON (remaining: %.1fs)", remaining.Seconds())
	}

	p.logger.Info("Starting battery startup sequence",
		"current_mode", p.currentMode,
		"current_shutdown", p.shutdownState)

	// Step 1: Write Shutdown=0 (active)
	err := p.client.WriteMetric(shutdownCommandBlock.Metrics["Shutdown"], uint16(0))
	if err != nil {
		return fmt.Errorf("startup - write Shutdown=0: %w", err)
	}

	// Update cached shutdown state immediately after successful write
	p.shutdownState = 0

	// Step 2: Start 60s safety period with pending Mode=1 write
	safetyEnd := time.Now().Add(60 * time.Second)
	p.safetyPeriodEnd = &safetyEnd
	p.pendingModeWrite = true // Flag to write Mode=1 after safety period

	p.logger.Info("Startup sequence initiated, 60s safety period started (Mode=1 pending)",
		"safety_period_end", safetyEnd.Format(time.RFC3339))

	return nil
}

// checkSafetyPeriod checks if the 60s safety period has expired and completes any pending operations.
// Called from Run() loop on every telemetry poll (every 2 seconds).
//
// If safety period expired and pendingModeWrite is true, writes Mode=1 + Timeout to complete startup.
//
// Returns error if ModbusTCP write fails. On error, keeps safety period active for retry.
func (p *PowerPack) checkSafetyPeriod() error {
	// No safety period active
	if p.safetyPeriodEnd == nil {
		return nil
	}

	// Still in safety period
	now := time.Now()
	if now.Before(*p.safetyPeriodEnd) {
		// Log remaining time periodically (every ~10 seconds)
		remaining := time.Until(*p.safetyPeriodEnd)
		if int(remaining.Seconds())%10 == 0 || remaining.Seconds() < 5 {
			p.logger.Debug("Safety period in progress",
				"remaining_secs", remaining.Seconds(),
				"pending_mode_write", p.pendingModeWrite)
		}
		return nil
	}

	// Safety period expired - complete any pending operations
	if p.pendingModeWrite {
		p.logger.Info("Safety period expired, completing startup (writing Mode=1)")

		// Sanity check: verify battery state before writing Mode=1
		if p.shutdownState == 1 {
			// Battery is shutdown - something external happened
			p.logger.Warn("Safety period expired but battery is shutdown - aborting pending Mode=1 write",
				"current_shutdown", p.shutdownState)
			p.safetyPeriodEnd = nil
			p.pendingModeWrite = false
			return nil
		}

		// Write Mode=1 (Direct)
		err := p.client.WriteMetric(realPowerCommandBlock.Metrics["Mode"], uint16(1))
		if err != nil {
			// Keep safety period active to retry next poll
			return fmt.Errorf("complete startup - write Mode=1: %w", err)
		}

		// Write Timeout
		err = p.client.WriteMetric(directRealPowerCommandBlock.Metrics["Timeout"], MODBUS_TIMEOUT_SECS)
		if err != nil {
			// Mode=1 succeeded but timeout failed - log warning but continue
			p.logger.Warn("Failed to write timeout after Mode=1", "error", err)
		}

		p.logger.Info("Startup sequence complete (Mode=1 written)")
		p.pendingModeWrite = false
	} else {
		p.logger.Info("Safety period complete (shutdown sequence finished)")
	}

	// Clear safety period
	p.safetyPeriodEnd = nil
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
