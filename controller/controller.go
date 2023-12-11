package controller

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/cepro/besscontroller/config"
	"github.com/cepro/besscontroller/telemetry"
	timeutils "github.com/cepro/besscontroller/time_utils"
)

// Controller manages the power/energy levels of a BESS.
//
// It supports the following modes:
// Import Avoidance: discharges power into the microgrid if it detects an import from the national grid.
// Export Avoidance: takes power from the microgrid if it detects an export onto the national grid.
// Charge to min: If the SoE of the battery is below a minimum then it is charged up to that minimum.
//
// Put new site meter and bess readings onto the `SiteMeterReadings` and `BessReadings` channels. Instruction commands for
// the BESS will be output onto the `BessCommands` channel.
type Controller struct {
	SiteMeterReadings chan telemetry.MeterReading
	BessReadings      chan telemetry.BessReading

	config Config

	sitePower timedMetric
	bessSoe   timedMetric

	lastTargetPower float64
}

type PeriodWithSoe struct {
	Period timeutils.Period
	Soe    float64
}

type Config struct {
	BessNameplatePower   float64 // power capability of the bess in kW
	BessNameplateEnergy  float64 // energy storage capability of the bess in kW
	BessIsEmulated       bool    // If true, the site meter readings are artificially adjusted to account for the lack of real BESS import/export.
	BessSoeMin           float64 // The minimum SoE that the BESS will be allowed to fall to
	BessSoeMax           float64 // The maximum SoE that the BESS will be allowed to charge to
	BessChargeEfficiency float64 // Value from 0.0 to 1.0 giving the efficiency of charging

	ImportAvoidancePeriods []timeutils.ClockTimePeriod     // the periods of time to activate 'import avoidance'
	ExportAvoidancePeriods []timeutils.ClockTimePeriod     // the periods of time to activate 'export avoidance'
	ChargeToMinPeriods     []config.ClockTimePeriodWithSoe // the periods of time to recharge the battery, and the minimum level that the battery should be recharged to

	MaxReadingAge time.Duration // the maximum age of telemetry data before it's considered too stale to operate on, and the controller is stopped until new readings are available

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
		"bess_soe_min", c.config.BessSoeMin,
		"bess_soe_max", c.config.BessSoeMax,
		"bess_charge_efficiency", c.config.BessChargeEfficiency,
		"ctrl_import_avoidance_periods", fmt.Sprintf("%+v", c.config.ImportAvoidancePeriods),
		"ctrl_export_avoidance_periods", fmt.Sprintf("%+v", c.config.ExportAvoidancePeriods),
		"charge_to_min_periods", fmt.Sprintf("%+v", c.config.ChargeToMinPeriods),
	)

	slog.Info("Controller running")
	for {
		select {
		case <-ctx.Done():
			return
		case t := <-tickerChan:
			if c.sitePower.isOlderThan(c.config.MaxReadingAge) {
				slog.Error("Site power reading is too old to use, skipping this control loop.", "data_updated_at", c.sitePower.updatedAt, "data_max_age", c.config.MaxReadingAge)
				continue
			}
			if c.bessSoe.isOlderThan(c.config.MaxReadingAge) {
				slog.Error("BESS SoE reading is too old to use, skipping this control loop.", "data_updated_at", c.sitePower.updatedAt, "data_max_age", c.config.MaxReadingAge)
				continue
			}

			c.runControlLoop(t)
		case reading := <-c.SiteMeterReadings:
			if reading.PowerTotalActive == nil {
				slog.Error("No active power available in site meter reading")
				continue
			}
			c.sitePower.set(*reading.PowerTotalActive)
		case reading := <-c.BessReadings:
			c.bessSoe.set(reading.Soe)
		}
	}
}

func (c *Controller) EmulatedSitePower() float64 {
	// If the BESS is emulated then it cannot actually export or import power, and so it cannot actually effect the site meter readings.
	// Without the effect of the BESS on the site meter readings there is no 'closed loop control' and so if there is any site import the
	// controller will increase BESS output to the maximum and empty the battery.
	// So here we mock the effect that the BESS would have had on the site meter reading as if it was real.
	return c.sitePower.value - c.lastTargetPower
}

// runControlLoop inspects the latest telemetry and attempts to bring the microgrid site power level to <=0 during
// 'import avoidance periods' by setting the battery power level appropriately.
func (c *Controller) runControlLoop(t time.Time) {

	sitePower := c.sitePower.value
	if c.config.BessIsEmulated {
		sitePower = c.EmulatedSitePower()
	}

	// Calculate the different control signals from the different modes of operation
	importAvoidancePower, importAvoidanceActive := importAvoidance(t, c.config.ImportAvoidancePeriods, sitePower, c.lastTargetPower)
	exportAvoidancePower, exportAvoidanceActive := exportAvoidance(t, c.config.ExportAvoidancePeriods, sitePower, c.lastTargetPower)
	chargeToMinPower, chargeToMinActive := chargeToMin(t, c.config.ChargeToMinPeriods, c.bessSoe.value, c.config.BessChargeEfficiency)

	// Calculate the target power for the BESS by applying a priority order to the different control signals
	targetPower := 0.0
	if chargeToMinActive {
		targetPower = chargeToMinPower
	} else if exportAvoidanceActive {
		targetPower = exportAvoidancePower
	} else if importAvoidanceActive {
		targetPower = importAvoidancePower
	}

	// Dont try to exceed the nameplate power of the BESS
	if targetPower > c.config.BessNameplatePower {
		targetPower = c.config.BessNameplatePower
	}
	if targetPower < -c.config.BessNameplatePower {
		targetPower = -c.config.BessNameplatePower
	}

	// Dont exceed the min/max SoE limits
	soeMinActive := false
	soeMaxActive := false
	if targetPower > 0 && c.bessSoe.value <= c.config.BessSoeMin {
		soeMinActive = true
		targetPower = 0
	}
	if targetPower < 0 && c.bessSoe.value >= c.config.BessSoeMax {
		soeMaxActive = true
		targetPower = 0
	}

	slog.Info(
		"Controlling BESS",
		"site_power", c.sitePower.value,
		"bess_soe", c.bessSoe.value,
		"bess_soe_min_active", soeMinActive,
		"bess_soe_max_active", soeMaxActive,
		"import_avoidance_active", importAvoidanceActive,
		"export_avoidance_active", exportAvoidanceActive,
		"charge_to_min_active", chargeToMinActive,
		"bess_last_target_power", c.lastTargetPower,
		"bess_target_power", targetPower,
	)

	command := telemetry.BessCommand{
		TargetPower: targetPower,
	}
	sendIfNonBlocking(c.config.BessCommands, command, "PowerPack commands")
	c.lastTargetPower = targetPower
}

// importAvoidance returns the power level that should be applied to the battery for import avoidance, and a boolean indicating if import avoidance is active.
func importAvoidance(t time.Time, importAvoidancePeriods []timeutils.ClockTimePeriod, sitePower, lastTargetPower float64) (float64, bool) {

	importAvoidancePeriod := periodContainingTime(t, importAvoidancePeriods)
	if importAvoidancePeriod == nil {
		return 0.0, false
	}

	importAvoidancePower := sitePower + lastTargetPower
	if importAvoidancePower < 0 {
		return 0.0, false
	}

	return importAvoidancePower, true
}

// exportAvoidance returns the power level that should be applied to the battery for import avoidance, and a boolean indicating if import avoidance is active.
func exportAvoidance(t time.Time, exportAvoidancePeriods []timeutils.ClockTimePeriod, sitePower, lastTargetPower float64) (float64, bool) {

	exportAvoidancePeriod := periodContainingTime(t, exportAvoidancePeriods)
	if exportAvoidancePeriod == nil {
		return 0.0, false
	}

	exportAvoidancePower := sitePower + lastTargetPower
	if exportAvoidancePower > 0 {
		return 0.0, false
	}

	return exportAvoidancePower, true
}

// chargeToMin returns the power level that should be applied to the battery for "charge to min" functionality, and a boolean indicating if "charge to min" is active.
func chargeToMin(t time.Time, chargeToMinPeriods []config.ClockTimePeriodWithSoe, bessSoe, chargeEfficiency float64) (float64, bool) {

	periodWithSoe := periodWithSoeContainingTime(t, chargeToMinPeriods)
	if periodWithSoe == nil {
		return 0.0, false
	}

	// charge the battery to reach the minimum target SoE at the end of the period. If the battery is already charged to the minimum level then do nothing.
	energyToRecharge := (periodWithSoe.Soe - bessSoe) / chargeEfficiency
	if energyToRecharge <= 0 {
		return 0.0, false
	}

	durationToRecharge := periodWithSoe.Period.End.Sub(t)
	chargeToMinPower := -energyToRecharge / durationToRecharge.Hours()
	if chargeToMinPower > 0 {
		return 0.0, false
	}

	return chargeToMinPower, true
}

// periodWithSoeContainingTime returns the PeriodWithSoe that overlaps the given time if there is one, otherwise it returns nil.
// If there is more than one overlapping period than the first is returned.
func periodWithSoeContainingTime(t time.Time, ctPeriodsWithSoe []config.ClockTimePeriodWithSoe) *PeriodWithSoe {
	for _, ctPeriodWithSoe := range ctPeriodsWithSoe {
		period, ok := ctPeriodWithSoe.Period.AbsolutePeriod(t)
		if ok {
			return &PeriodWithSoe{
				Period: period,
				Soe:    ctPeriodWithSoe.Soe,
			}
		}
	}
	return nil
}

// periodContainingTime returns the first period that overlaps the given time if there is one, otherwise it returns nil.
// If there is more than one overlapping period than the first is returned.
func periodContainingTime(t time.Time, clockTimePeriods []timeutils.ClockTimePeriod) *timeutils.Period {
	for _, clockTimePeriod := range clockTimePeriods {
		period, ok := clockTimePeriod.AbsolutePeriod(t)
		if ok {
			return &period
		}
	}
	return nil
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
