package controller

import (
	"context"
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
// Put new site meter and bess readings onto the appropriate channels. Instruction commands for the BESS will
// be output onto the `BessCommands` channel.
type Controller struct {
	SiteMeterReadings chan telemetry.MeterReading
	BessReadings      chan telemetry.BessReading

	config Config

	sitePower float64
	bessSoe   float64

	lastTargetPower float64
}

type Config struct {
	BessNameplatePower  float64
	BessNameplateEnergy float64
	BessMinimumSoE      float64 // The minimum state of energy that the battery should hit

	ImportAvoidancePeriods []timeutils.ClockTimePeriod

	BessCommands chan<- telemetry.BessCommand
}

func New(config Config) *Controller {
	return &Controller{
		SiteMeterReadings: make(chan telemetry.MeterReading),
		BessReadings:      make(chan telemetry.BessReading),
		config:            config,
	}
}

// Run loops forever, running the control loop every 5 seconds, and storing any neccesary data from the available meter and
// bess readings whenever they become available.
func (c *Controller) Run(ctx context.Context) {
	ctrlLoopTicker := time.NewTicker(time.Second * 5)

	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ctrlLoopTicker.C:
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

	inImportAvoidancePeriod := false
	for _, period := range c.config.ImportAvoidancePeriods {
		if period.Contains(t) {
			inImportAvoidancePeriod = true
			break
		}
	}

	targetPower := 0.0
	if inImportAvoidancePeriod && c.sitePower > 0 && c.bessSoe > c.config.BessMinimumSoE {
		targetPower = c.sitePower + c.lastTargetPower
	}

	if targetPower > c.config.BessNameplatePower {
		targetPower = c.config.BessNameplatePower
	}
	c.config.BessCommands <- telemetry.BessCommand{
		TargetPower: targetPower,
	}
	c.lastTargetPower = targetPower
	slog.Info(
		"Controlling BESS",
		"site_power", c.sitePower,
		"bess_soe", c.bessSoe,
		"import_avoidance_active", inImportAvoidancePeriod,
		"bess_target_power", targetPower,
	)
}
