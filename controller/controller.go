package controller

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/cepro/besscontroller/cartesian"
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

	lastBessTargetPower float64
}

type Config struct {
	BessNameplatePower   float64 // power capability of the bess in kW
	BessNameplateEnergy  float64 // energy storage capability of the bess in kW
	BessIsEmulated       bool    // If true, the site meter readings are artificially adjusted to account for the lack of real BESS import/export.
	BessSoeMin           float64 // The minimum SoE that the BESS will be allowed to fall to
	BessSoeMax           float64 // The maximum SoE that the BESS will be allowed to charge to
	BessChargeEfficiency float64 // Value from 0.0 to 1.0 giving the efficiency of charging

	SiteImportPowerLimit float64 // Max power that can be imported from the microgrid boundary
	SiteExportPowerLimit float64 // Max power that can be exported from the microgrid boundary

	ImportAvoidancePeriods []timeutils.ClockTimePeriod     // the periods of time to activate 'import avoidance'
	ExportAvoidancePeriods []timeutils.ClockTimePeriod     // the periods of time to activate 'export avoidance'
	ChargeToMinPeriods     []config.ClockTimePeriodWithSoe // the periods of time to recharge the battery, and the minimum level that the battery should be recharged to
	NivChasePeriods        []timeutils.ClockTimePeriod     // the periods of time to activate 'niv chasing'

	NivChargeCurve    cartesian.Curve // Price/SoE curve to charge to when NIV chasing
	NivDischargeCurve cartesian.Curve // Price/SoE curve to discharge to when NIV chasing
	DuosChargesImport []DuosCharge
	DuosChargesExport []DuosCharge

	ModoClient imbalancePricer

	MaxReadingAge time.Duration // the maximum age of telemetry data before it's considered too stale to operate on, and the controller is stopped until new readings are available

	BessCommands chan<- telemetry.BessCommand // Channel that bess control commands will be sent to
}

type PeriodWithSoe struct {
	Period timeutils.Period
	Soe    float64
}

// imbalancePricer is an interface onto any object that provides imbalance pricing
type imbalancePricer interface {
	ImbalancePrice() (float64, time.Time)
}

// controlComponent represents the output of some control mode - e.g. export avoidance or NIV chasing etc
type controlComponent struct {
	name         string       // Friendly name of this component for debug logging
	isActive     bool         // Set True if this control component is activated, or false if it is not doing anything
	targetPower  float64      // The power associated with this control component
	controlPoint controlPoint // Where the power should be applied
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
		"niv_chase_periods", fmt.Sprintf("%+v", c.config.NivChasePeriods),
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
	return c.sitePower.value - c.lastBessTargetPower
}

func (c *Controller) SitePower() float64 {
	if c.config.BessIsEmulated {
		return c.EmulatedSitePower()
	}
	return c.sitePower.value
}

// runControlLoop inspects the latest telemetry and controls the battery according to the highest priority control component.
func (c *Controller) runControlLoop(t time.Time) {

	// DUoS charges change depending on the time of day - get the current charges
	duosChargeImport := getDuosCharges(t, c.config.DuosChargesImport)
	duosChargeExport := getDuosCharges(t, c.config.DuosChargesExport)

	// Calculate the different control components from the different modes of operation, listed in priority order
	components := []controlComponent{
		nivChase(
			t,
			c.config.NivChasePeriods,
			c.config.NivChargeCurve,
			c.config.NivDischargeCurve,
			c.bessSoe.value,
			c.config.BessChargeEfficiency,
			duosChargeImport,
			duosChargeExport,
			c.config.ModoClient,
		),
		chargeToMin(
			t,
			c.config.ChargeToMinPeriods,
			c.bessSoe.value,
			c.config.BessChargeEfficiency,
		),
		importAvoidance(
			t,
			c.config.ImportAvoidancePeriods,
			c.SitePower(),
			c.lastBessTargetPower,
		),
		exportAvoidance(
			t,
			c.config.ExportAvoidancePeriods,
			c.SitePower(),
			c.lastBessTargetPower,
		),
	}

	// Select the highest priority component that is active, or default to a zero-power "idle" component
	component := controlComponent{
		name:         "idle",
		isActive:     true,
		targetPower:  0,
		controlPoint: controlPointBess,
	}
	for i := range components {
		if components[i].isActive {
			component = components[i]
			break
		}
	}

	bessTargetPower, limits := c.calculateBessPower(component)

	slog.Info(
		"Controlling BESS",
		"site_power", c.sitePower.value,
		"bess_soe", c.bessSoe.value,
		"control_component", component.name,
		"control_component_power", component.targetPower,
		"control_component_point", component.controlPoint,
		"site_power_limits_active", limits.sitePower,
		"bess_power_limits_active", limits.bessPower,
		"bess_soe_limits_active", limits.bessSoe,
		"duos_charge_import", duosChargeImport,
		"duos_charge_export", duosChargeExport,
		"bess_last_target_power", c.lastBessTargetPower,
		"bess_target_power", bessTargetPower,
	)

	command := telemetry.BessCommand{
		TargetPower: bessTargetPower,
	}
	sendIfNonBlocking(c.config.BessCommands, command, "PowerPack commands")
	c.lastBessTargetPower = bessTargetPower
}

// activeLimits provides debug information on which limits were used in the calculation of the BESS power level.
type activeLimits struct {
	bessPower bool
	sitePower bool
	bessSoe   bool
}

// calculateBessPower returns the power level that should be sent to the BESS, given the control component. Limits are applied to keep the SoE, BESS power, and
// site power within bounds. Details of which limits were activated in the calculation are returned.
func (c *Controller) calculateBessPower(component controlComponent) (float64, activeLimits) {

	var bessPowerLimitsActive1 bool
	var bessPowerLimitsActive2 bool
	var sitePowerLimitsActive bool
	var bessSoeLimitActive bool

	if component.controlPoint == controlPointBess {

		// Apply the physical power limits of the BESS
		component.targetPower, bessPowerLimitsActive1 = limitValue(component.targetPower, c.config.BessNameplatePower, c.config.BessNameplatePower)

		// If we want to control the power at the BESS inverter meter, then we must ensure that it will not exceed the site connection limits.
		// If it will exceed the site limits, then we move the control point to the site level, and apply the site connection limits.
		bessPowerDiff := component.targetPower - c.lastBessTargetPower
		expectedSitePower := c.SitePower() - bessPowerDiff // Site power: positive is import, negative is export. Battery power: positive is discharge, negative is charge.
		limitedSitePower, sitePowerLimitsActive := limitValue(expectedSitePower, c.config.SiteImportPowerLimit, c.config.SiteExportPowerLimit)
		if sitePowerLimitsActive {
			component.controlPoint = controlPointSite
			component.targetPower = limitedSitePower
		}
	} else if component.controlPoint == controlPointSite {

		// If we want to control the power at the site meter, then apply the site connection limits:
		component.targetPower, sitePowerLimitsActive = limitValue(component.targetPower, c.config.SiteImportPowerLimit, c.config.SiteExportPowerLimit)

	} else {
		panic(fmt.Sprintf("Unknown control point: %v", component.controlPoint))
	}

	bessTargetPower := 0.0
	if component.controlPoint == controlPointSite {
		// Convert the requested site power control level into a bess power control level
		err := component.targetPower - c.SitePower()
		bessTargetPower = c.lastBessTargetPower - err
	} else if component.controlPoint == controlPointBess {
		bessTargetPower = component.targetPower
	} else {
		panic(fmt.Sprintf("Unknown control point: %v", component.controlPoint))
	}

	// Re-apply the BESS power limits here as it's possible that the conversion from site power control level to bess power control level may have produced
	// a target power outside of the BESS' capabilities
	bessTargetPower, bessPowerLimitsActive2 = limitValue(bessTargetPower, c.config.BessNameplatePower, c.config.BessNameplatePower)

	// Apply BESS SoE limits
	if bessTargetPower > 0 && c.bessSoe.value <= c.config.BessSoeMin {
		bessTargetPower = 0
		bessSoeLimitActive = true
	}
	if bessTargetPower < 0 && c.bessSoe.value >= c.config.BessSoeMax {
		bessTargetPower = 0
		bessSoeLimitActive = true
	}

	return bessTargetPower, activeLimits{
		bessPower: bessPowerLimitsActive1 || bessPowerLimitsActive2,
		sitePower: sitePowerLimitsActive,
		bessSoe:   bessSoeLimitActive,
	}
}

// limitValue returns the value capped between `maxPositive` and `maxNegative`, alongside a boolean indicating if limits needed to be applied
func limitValue(value, maxPositive, maxNegative float64) (float64, bool) {
	if value > maxPositive {
		return maxPositive, true
	} else if value < -maxNegative {
		return -maxNegative, true
	} else {
		return value, false
	}
}

// importAvoidance returns control component for avoiding site imports.
func importAvoidance(t time.Time, importAvoidancePeriods []timeutils.ClockTimePeriod, sitePower, lastTargetPower float64) controlComponent {

	importAvoidancePeriod := periodContainingTime(t, importAvoidancePeriods)
	if importAvoidancePeriod == nil {
		return controlComponent{isActive: false}
	}

	importAvoidancePower := sitePower + lastTargetPower
	if importAvoidancePower < 0 {
		return controlComponent{isActive: false}
	}

	return controlComponent{
		name:         "import_avoidance",
		isActive:     true,
		targetPower:  0, // Target zero power at the site boundary
		controlPoint: controlPointSite,
	}
}

// exportAvoidance returns the control component for avoiding site exports.
func exportAvoidance(t time.Time, exportAvoidancePeriods []timeutils.ClockTimePeriod, sitePower, lastTargetPower float64) controlComponent {

	exportAvoidancePeriod := periodContainingTime(t, exportAvoidancePeriods)
	if exportAvoidancePeriod == nil {
		return controlComponent{isActive: false}
	}

	exportAvoidancePower := sitePower + lastTargetPower
	if exportAvoidancePower > 0 {
		return controlComponent{isActive: false}
	}

	return controlComponent{
		name:         "export_avoidance",
		isActive:     true,
		targetPower:  0, // Target zero power at the site boundary
		controlPoint: controlPointSite,
	}
}

// chargeToMin returns the control component for charging the battery to a minimum SoE.
func chargeToMin(t time.Time, chargeToMinPeriods []config.ClockTimePeriodWithSoe, bessSoe, chargeEfficiency float64) controlComponent {

	periodWithSoe := periodWithSoeContainingTime(t, chargeToMinPeriods)
	if periodWithSoe == nil {
		return controlComponent{isActive: false}
	}

	// charge the battery to reach the minimum target SoE at the end of the period. If the battery is already charged to the minimum level then do nothing.
	energyToRecharge := (periodWithSoe.Soe - bessSoe) / chargeEfficiency
	if energyToRecharge <= 0 {
		return controlComponent{isActive: false}
	}

	durationToRecharge := periodWithSoe.Period.End.Sub(t)
	chargeToMinPower := -energyToRecharge / durationToRecharge.Hours()
	if chargeToMinPower > 0 {
		return controlComponent{isActive: false}
	}

	return controlComponent{
		name:         "charge_to_min",
		isActive:     true,
		targetPower:  chargeToMinPower,
		controlPoint: controlPointBess,
	}
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
