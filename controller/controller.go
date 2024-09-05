package controller

import (
	"context"
	"fmt"
	"log/slog"
	"math"
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
// Import Avoidance when short: discharges into the microgrid if it detects an import from the national grid AND the system is short (prices are likely high)
// Charge to SoE: If the SoE of the battery is below a minimum then it is charged up to that minimum.
// Discharge to SoE: If the SoE of the battery is above a maximum then it is charged up to that maximum.
// Niv chasing: the imbalance price is used to influence charge/discharges
//
// Put new site meter and bess readings onto the `SiteMeterReadings` and `BessReadings` channels. Instruction commands for
// the BESS will be output onto the `BessCommands` channel (supplied via the Config).
type Controller struct {
	SiteMeterReadings chan telemetry.MeterReading
	BessReadings      chan telemetry.BessReading

	config Config

	sitePower timedMetric // +ve is microgrid import, -ve is microgrid export
	bessSoe   timedMetric

	lastBessTargetPower float64 // +ve is battery discharge, -ve is battery charge
}

type Config struct {
	BessIsEmulated          bool    // If true, the site meter readings are artificially adjusted to account for the lack of real BESS import/export.
	BessChargeEfficiency    float64 // Value from 0.0 to 1.0 giving the efficiency of charging
	BessSoeMin              float64 // The minimum SoE that the BESS will be allowed to fall to
	BessSoeMax              float64 // The maximum SoE that the BESS will be allowed to charge to
	BessChargePowerLimit    float64 // The maximum power that we can call on the BESS to charge at
	BessDischargePowerLimit float64 // The maximum power that we can call on the BESS to discharge at
	SiteImportPowerLimit    float64 // Max power that can be imported from the microgrid boundary
	SiteExportPowerLimit    float64 // Max power that can be exported from the microgrid boundary

	// Configuration of the different modes of operation:
	ImportAvoidancePeriods   []timeutils.DayedPeriod                 // the periods of time to activate 'import avoidance'
	ExportAvoidancePeriods   []timeutils.DayedPeriod                 // the periods of time to activate 'export avoidance'
	ImportAvoidanceWhenShort []config.ImportAvoidanceWhenShortConfig // periods of time to activate 'import avoidance when short'
	ChargeToSoePeriods       []config.DayedPeriodWithSoe             // the periods of time to charge the battery, and the level that the battery should be recharged to
	DischargeToSoePeriods    []config.DayedPeriodWithSoe             // the periods of time to discharge the battery, and the level that the battery should be discharged to
	DynamicPeakDischarges    []config.DynamicPeakDischargeConfig     // the periods of time to discharge 'dynamically' into a peak
	NivChasePeriods          []config.DayedPeriodWithNIV             // the periods of time to activate 'niv chasing', and the associated configuraiton

	RatesImport []config.TimedRate // Any charges that apply to importing power from the grid
	RatesExport []config.TimedRate // Any charges that apply to exporting power from the grid

	ModoClient imbalancePricer

	MaxReadingAge time.Duration // the maximum age of telemetry data before it's considered too stale to operate on, and the controller is stopped until new readings are available

	BessCommands chan<- telemetry.BessCommand // Channel that bess control commands will be sent to
}

// imbalancePricer is an interface onto any object that provides imbalance pricing and volumes
type imbalancePricer interface {
	ImbalancePrice() (float64, time.Time)  // ImbalancePrice returns the last cached imbalance price, and the settlement period time that it corresponds to
	ImbalanceVolume() (float64, time.Time) // ImbalanceVolume returns the last cached imbalance volume, and the settlement period time that it corresponds to
}

// controlComponent represents the output of some control mode - e.g. export avoidance or NIV chasing etc
type controlComponent struct {
	name         string       // Friendly name of this component for debug logging
	isActive     bool         // Set True if this control component is activated, or False if it is not doing anything
	targetPower  float64      // The power associated with this control component
	controlPoint controlPoint // The point where the targetPower should be applied
}

// New creates a new Controller using the given Config
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

	// TODO: print all configuration here
	slog.Info(
		"Starting controller",
		"bess_soe_min", c.config.BessSoeMin,
		"bess_soe_max", c.config.BessSoeMax,
		"bess_charge_power_limit", c.config.BessChargePowerLimit,
		"bess_discharge_power_limit", c.config.BessDischargePowerLimit,
		"site_import_power_limit", c.config.SiteImportPowerLimit,
		"site_export_power_limit", c.config.SiteExportPowerLimit,
		"bess_charge_efficiency", c.config.BessChargeEfficiency,
		"import_avoidance_periods", fmt.Sprintf("%+v", c.config.ImportAvoidancePeriods),
		"export_avoidance_periods", fmt.Sprintf("%+v", c.config.ExportAvoidancePeriods),
		"import_avoidance_periods_when_short", fmt.Sprintf("%+v", c.config.ImportAvoidanceWhenShort),
		"charge_to_soe_periods", fmt.Sprintf("%+v", c.config.ChargeToSoePeriods),
		"discharge_to_soe_periods", fmt.Sprintf("%+v", c.config.DischargeToSoePeriods),
		"niv_chase_periods", fmt.Sprintf("%+v", c.config.NivChasePeriods),
		"rates_import", fmt.Sprintf("%+v", c.config.RatesImport),
		"rates_export", fmt.Sprintf("%+v", c.config.RatesExport),
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

// SitePower returns the metered power reading at the microgrid boundary (or an emulated value if appropriate)
func (c *Controller) SitePower() float64 {
	if c.config.BessIsEmulated {
		return c.EmulatedSitePower()
	}
	return c.sitePower.value
}

// runControlLoop inspects the latest telemetry and controls the battery according to the highest priority control component.
func (c *Controller) runControlLoop(t time.Time) {

	// Rates change depending on the time of day - get the current rates
	ratesImport := config.SumTimedRates(t, c.config.RatesImport)
	ratesExport := config.SumTimedRates(t, c.config.RatesExport)

	// Calculate the different control components from the different modes of operation, listed in priority order
	components := []controlComponent{
		dischargeToSoe(
			t,
			c.config.DischargeToSoePeriods,
			c.bessSoe.value,
			1.0, // Discharge efficiency is assumed to be 100%
		),
		dynamicPeakDischarge(
			t,
			c.config.DynamicPeakDischarges,
			c.bessSoe.value,
			c.SitePower(),
			c.lastBessTargetPower,
			c.maxBessDischarge(),
			c.config.ModoClient,
		),
		nivChase(
			t,
			c.config.NivChasePeriods,
			c.bessSoe.value,
			c.config.BessChargeEfficiency,
			ratesImport,
			ratesExport,
			c.config.ModoClient,
		),
		chargeToSoe(
			t,
			c.config.ChargeToSoePeriods,
			c.bessSoe.value,
			c.config.BessChargeEfficiency,
		),
		basicImportAvoidance(
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
		importAvoidanceWhenShort(
			t,
			c.config.ImportAvoidanceWhenShort,
			c.SitePower(),
			c.lastBessTargetPower,
			c.config.ModoClient,
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
		"rates_import", ratesImport,
		"rates_export", ratesExport,
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
		component.targetPower, bessPowerLimitsActive1 = limitValue(component.targetPower, c.config.BessDischargePowerLimit, c.config.BessChargePowerLimit)

		// If we want to control the power at the BESS inverter meter, then we must ensure that it will not exceed the site connection limits.
		// If it will exceed the site limits, then we move the control point to the site level, and apply the site connection limits.
		bessPowerDiff := component.targetPower - c.lastBessTargetPower
		expectedSitePower := c.SitePower() - bessPowerDiff // Site power: positive is import, negative is export. Battery power: positive is discharge, negative is charge.
		var limitedSitePower = 0.0
		limitedSitePower, sitePowerLimitsActive = limitValue(expectedSitePower, c.config.SiteImportPowerLimit, c.config.SiteExportPowerLimit)
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
	bessTargetPower, bessPowerLimitsActive2 = limitValue(bessTargetPower, c.config.BessDischargePowerLimit, c.config.BessChargePowerLimit)

	// TODO: there are some edge-case scenarios where the sign of the targetPower could change - e.g. if solar exports exceed the site limits.
	// In that scenario we might just want to turn the battery off?

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

// maxBessDischarge returns the maximum discharge rate of the BESS at this point in time.
func (c *Controller) maxBessDischarge() float64 {
	// Use a contrived controlComponent that requests infinite discharge power to get the actual maximum rate with the existing `calculateBessPower` method.
	maxBessDischarge, _ := c.calculateBessPower(controlComponent{
		"max_discharge_calculation",
		true,
		math.Inf(-1),
		controlPointBess,
	})
	return maxBessDischarge
}

// importAvoidanceWhenShort returns control component for avoiding site imports, based on imbalance status
func importAvoidanceWhenShort(t time.Time, configs []config.ImportAvoidanceWhenShortConfig, sitePower, lastTargetPower float64, modoClient imbalancePricer) controlComponent {

	conf, _ := findPeriodicalConfigForTime(t, configs)
	if conf == nil {
		return controlComponent{isActive: false}
	}

	_, imbalanceVolume, gotPrediction := predictImbalance(
		t,
		config.NivPredictionConfig{
			WhenShort: conf.ShortPrediction,
			// We are only interested in the case where the system is short, so don't allow predictions for long
			WhenLong: config.NivPredictionDirectionConfig{AllowPrediction: false},
		},
		modoClient,
	)
	if !gotPrediction {
		// We don't have any pricing data available, so do nothing
		return controlComponent{isActive: false}
	}

	if imbalanceVolume <= 0 {
		// We aren't short, so do nothing
		return controlComponent{isActive: false}
	}

	return importAvoidanceHelper(sitePower, lastTargetPower, "import_avoidance_when_short")
}

// basicImportAvoidance returns the control component for avoiding microgrid boundary imports.
func basicImportAvoidance(t time.Time, importAvoidancePeriods []timeutils.DayedPeriod, sitePower, lastTargetPower float64) controlComponent {

	_, importAvoidancePeriod := findDayedPeriodContainingTime(t, importAvoidancePeriods)
	if importAvoidancePeriod == nil {
		return controlComponent{isActive: false}
	}

	return importAvoidanceHelper(sitePower, lastTargetPower, "import_avoidance")
}

// importAvoidanceHelper generates the control component for an import avoidance action.
// Import avoidance is a strategy that is used by a few different control modes so this is a conveninence function for helping with that.
func importAvoidanceHelper(sitePower, lastTargetPower float64, controlComponentName string) controlComponent {
	importAvoidancePower := sitePower + lastTargetPower
	if importAvoidancePower < 0 {
		return controlComponent{isActive: false}
	}

	return controlComponent{
		name:         controlComponentName,
		isActive:     true,
		targetPower:  0, // Target zero power at the site boundary
		controlPoint: controlPointSite,
	}
}

// exportAvoidance returns the control component for avoiding microgrid boundary exports.
func exportAvoidance(t time.Time, exportAvoidancePeriods []timeutils.DayedPeriod, sitePower, lastTargetPower float64) controlComponent {

	_, exportAvoidancePeriod := findDayedPeriodContainingTime(t, exportAvoidancePeriods)
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

// chargeToSoe returns the control component for charging the battery to a minimum SoE.
func chargeToSoe(t time.Time, configs []config.DayedPeriodWithSoe, bessSoe, chargeEfficiency float64) controlComponent {

	conf, absPeriod := findPeriodicalConfigForTime(t, configs)
	if conf == nil {
		return controlComponent{isActive: false}
	}

	targetSoe := conf.Soe
	endOfCharge := absPeriod.End

	// charge the battery to reach the minimum target SoE at the end of the period. If the battery is already charged to the minimum level then do nothing.
	energyToCharge := (targetSoe - bessSoe) / chargeEfficiency
	if energyToCharge <= 0 {
		return controlComponent{isActive: false}
	}

	durationToRecharge := endOfCharge.Sub(t)
	chargePower := -energyToCharge / durationToRecharge.Hours()
	if chargePower >= 0 {
		return controlComponent{isActive: false}
	}

	return controlComponent{
		name:         "charge_to_soe",
		isActive:     true,
		targetPower:  chargePower,
		controlPoint: controlPointBess,
	}
}

// dischargeToSoe returns the control component for discharging the battery to a pre-defined state of energy.
func dischargeToSoe(t time.Time, configs []config.DayedPeriodWithSoe, bessSoe, dischargeEfficiency float64) controlComponent {

	conf, absPeriod := findPeriodicalConfigForTime(t, configs)
	if conf == nil {
		return controlComponent{isActive: false}
	}

	targetSoe := conf.Soe
	endOfDischarge := absPeriod.End

	// discharge the battery to reach the target SoE at the end of the period. If the battery is already discharged to the target level, or below then do nothing.
	energyToDischarge := (bessSoe - targetSoe) * dischargeEfficiency
	if energyToDischarge <= 0 {
		return controlComponent{isActive: false}
	}

	durationToDischarge := endOfDischarge.Sub(t)
	dichargePower := energyToDischarge / durationToDischarge.Hours()
	if dichargePower <= 0 {
		return controlComponent{isActive: false}
	}

	return controlComponent{
		name:         "discharge_to_soe",
		isActive:     true,
		targetPower:  dichargePower,
		controlPoint: controlPointBess,
	}
}

// PeriodicalConfigTypes is an interface onto configuration structures that are tied to a particular periods of time
type PeriodicalConfigTypes interface {
	config.ImportAvoidanceWhenShortConfig | config.DayedPeriodWithSoe | config.DayedPeriodWithNIV | config.DynamicPeakDischargeConfig
	GetDayedPeriod() timeutils.DayedPeriod
}

// findPeriodicalConfigForTime searches the list of configs and returns the first one that is active at the time `t` (i.e.
// the first one whose period contains `t`), or nil if none was found. It also returns the associated absolute period.
func findPeriodicalConfigForTime[T PeriodicalConfigTypes](t time.Time, configs []T) (*T, timeutils.Period) {

	for _, conf := range configs {
		dayedPeriod := conf.GetDayedPeriod()
		absPeriod, ok := dayedPeriod.AbsolutePeriod(t)
		if ok {
			return &conf, absPeriod
		}
	}
	return nil, timeutils.Period{}
}

// findDayedPeriodContainingTime searches the list of dayed periods and returns the first one that is active at the time `t` (i.e.
// the first one whose period contains `t`), or nil if none was found. It also returns the associated absolute period.
func findDayedPeriodContainingTime(t time.Time, dayedPeriods []timeutils.DayedPeriod) (*timeutils.DayedPeriod, *timeutils.Period) {

	for _, dayedPeriod := range dayedPeriods {
		period, ok := dayedPeriod.AbsolutePeriod(t)
		if ok {
			return &dayedPeriod, &period
		}
	}
	return nil, nil
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
