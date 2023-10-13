package controller

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/cepro/besscontroller/telemetry"
	timeutils "github.com/cepro/besscontroller/time_utils"
)

// Controller manages the power/energy levels of a BESS.
//
// At the moment it only supports basic 'import avoidance' during the given `ImportAvoidancePeriods` whereby
// it discharges power into the microgrid if it detects an import from the national grid during a designated time period.
//
// Put new site meter and bess readings onto the `SiteMeterReadings` and `BessReadings` channels. Instruction commands for
// the BESS will be output onto the `BessCommands` channel.
type Controller struct {
	SiteMeterReadings chan telemetry.MeterReading
	BessReadings      chan telemetry.BessReading

	config Config

	sitePower float64
	bessSoe   float64

	lastTargetPower float64
}

type Config struct {
	BessNameplatePower  float64 // power capability of the bess in kW
	BessNameplateEnergy float64 // energy storage capability of the bess in kW
	BessMinSoe          float64 // The minimum state of energy that the battery should hit
	BessMaxSoe          float64 // The maximum state of energy that the battery should hit

	ImportAvoidancePeriods []timeutils.ClockTimePeriod // the periods of time to activate 'import avoidance'

	BessCommands chan<- telemetry.BessCommand // Channel that bess control commands will be sent to
}

func New(config Config) *Controller {
	return &Controller{
		SiteMeterReadings: make(chan telemetry.MeterReading, 1),
		BessReadings:      make(chan telemetry.BessReading, 1),
		config:            config,
	}
}

// Run loops forever, storing data from the available meter and bess readings whenever they become available, and running the control
// loop every time a tick is recieved on `tickerChan`.
func (c *Controller) Run(ctx context.Context, tickerChan <-chan time.Time) {

	slog.Info(
		"Starting controller",
		"bess_nameplate_power", c.config.BessNameplatePower,
		"bess_nameplate_energy", c.config.BessNameplateEnergy,
		"ctrl_import_avoidance_periods", fmt.Sprintf("%+v", c.config.ImportAvoidancePeriods),
	)

	slog.Info("Controller running")
	for {
		select {
		case <-ctx.Done():
			return
		case t := <-tickerChan:
			c.runControlLoop(t)
		case reading := <-c.SiteMeterReadings:
			c.sitePower = reading.PowerTotalActive
		case reading := <-c.BessReadings:
			c.bessSoe = reading.Soe
		}
	}
}

// runControlLoop inspects the latest telemetry and attempts to bring the microgrid site power level to <=0 during
// 'import avoidance periods' by setting the battery power level appropriately.
func (c *Controller) runControlLoop(t time.Time) {

	inImportAvoidancePeriod := timeIsInPeriods(t, c.config.ImportAvoidancePeriods)

	targetPower := 0.0

	belowSoEThreshold := c.bessSoe < c.config.BessMinSoe

	if inImportAvoidancePeriod {
		// discharge the battery if there is a site import

		if inImportAvoidancePeriod && !belowSoEThreshold {
			targetPower = c.sitePower + c.lastTargetPower
		}

		if targetPower < 0 {
			targetPower = 0
		}
		if targetPower > c.config.BessNameplatePower {
			targetPower = c.config.BessNameplatePower
		}
	} else {
		// charge the battery to reach BessMaxSoe before the next import avoidance period
		nextStartTimes := timeutils.NextStartTimes(t, c.config.ImportAvoidancePeriods)
		if len(nextStartTimes) < 1 {
			slog.Warn("No import avoidance periods, skipping rebalancing")
		} else {
			targetFullyChargedTime := nextStartTimes[0]
			energyToRecharge := c.config.BessMaxSoe - c.bessSoe
			durationToRecharge := targetFullyChargedTime.Sub(t)
			targetPower = -energyToRecharge / durationToRecharge.Hours()
		}
		if targetPower > 0 {
			targetPower = 0
		}
		if targetPower < -c.config.BessNameplatePower {
			targetPower = -c.config.BessNameplatePower
		}
	}

	slog.Info(
		"Controlling BESS",
		"bess_soe", c.bessSoe,
		"site_power", c.sitePower,
		"import_avoidance_active", inImportAvoidancePeriod,
		"bess_below_soe_threshold", belowSoEThreshold,
		"bess_last_target_power", c.lastTargetPower,
		"bess_target_power", targetPower,
	)

	command := telemetry.BessCommand{
		TargetPower: targetPower,
	}
	sendIfNonBlocking(c.config.BessCommands, command, "PowerPack commands")
	c.lastTargetPower = targetPower
}

// timeIsInPeriods returns true if the given time is within one of the given periods
func timeIsInPeriods(t time.Time, periods []timeutils.ClockTimePeriod) bool {
	for _, period := range periods {
		if period.Contains(t) {
			return true
		}
	}
	return false
}

// sendIfNonBlocking attempts to send the given value onto the given channel, but will only do so if the operation
// is non-blocking, otherwise it logs a warning message and returns.
func sendIfNonBlocking[V any](ch chan<- V, val V, messageTargetLogStr string) {
	select {
	case ch <- val:
	default:
		slog.Warn("Dropped message", "message_target", messageTargetLogStr)
	}
}
