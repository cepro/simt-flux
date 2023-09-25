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
	BessMinimumSoE      float64 // The minimum state of energy that the battery should hit

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
			slog.Debug("Running controller loop")
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

	// TODO: rebalance

	inImportAvoidancePeriod := false
	for _, period := range c.config.ImportAvoidancePeriods {
		if period.Contains(t) {
			inImportAvoidancePeriod = true
			break
		}
	}

	belowSoEThreshold := c.bessSoe < c.config.BessMinimumSoE

	targetPower := 0.0
	if inImportAvoidancePeriod && !belowSoEThreshold {
		targetPower = c.sitePower + c.lastTargetPower
	}

	if targetPower < 0 {
		targetPower = 0
	}
	if targetPower > c.config.BessNameplatePower {
		targetPower = c.config.BessNameplatePower
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

	c.config.BessCommands <- telemetry.BessCommand{
		TargetPower: targetPower,
	}
	c.lastTargetPower = targetPower
}
