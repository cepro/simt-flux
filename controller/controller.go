package controller

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/cepro/besscontroller/axle"
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
// Put new site meter and bess readings onto the `SiteMeterReadings` and `BessReadings` channels; put new schedules from Axle onto the `AxleSchedules`
// channel.
// Instruction commands for the BESS will be output onto the `BessCommands` channel (supplied via the Config).
type Controller struct {
	SiteMeterReadings chan telemetry.MeterReading
	BessReadings      chan telemetry.BessReading
	AxleSchedules     chan axle.Schedule

	config Config

	sitePower timedMetric // +ve is microgrid import, -ve is microgrid export
	bessSoe   timedMetric

	axleSchedule axle.Schedule

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
	DynamicPeakDischarges    []config.DynamicPeakDischargeConfig     // the periods of time to approach and discharge 'dynamically' into a peak
	DynamicPeakApproaches    []config.DynamicPeakApproachConfig      // the periods of time to approach and discharge 'dynamically' into a peak
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

// New creates a new Controller using the given Config
func New(config Config) *Controller {
	return &Controller{
		SiteMeterReadings: make(chan telemetry.MeterReading, 1),
		BessReadings:      make(chan telemetry.BessReading, 1),
		AxleSchedules:     make(chan axle.Schedule, 1),
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
		"dynamic_peak_discharges", fmt.Sprintf("%+v", c.config.DynamicPeakDischarges),
		"dynamic_peak_approaches", fmt.Sprintf("%+v", c.config.DynamicPeakApproaches),
		"niv_chase_periods", fmt.Sprintf("%+v", c.config.NivChasePeriods),
		"rates_import", fmt.Sprintf("%+v", c.config.RatesImport),
		"rates_export", fmt.Sprintf("%+v", c.config.RatesExport),
	)

	slog.Info("Controller running")
	for {
		select {
		case <-ctx.Done():
			return

		case reading := <-c.SiteMeterReadings:
			if reading.PowerTotalActive == nil {
				slog.Error("No active power available in site meter reading")
				continue
			}
			c.sitePower.set(*reading.PowerTotalActive)

		case reading := <-c.BessReadings:
			c.bessSoe.set(reading.Soe)

		case schedule := <-c.AxleSchedules:
			c.axleSchedule = schedule

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

	// Calculate the different control components that all the different modes of operation want to do now. These are listed inpriority order.
	components := []controlComponent{
		axleSchedule(
			t,
			c.axleSchedule,
			c.SitePower(),
			c.lastBessTargetPower,
		),
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
		dynamicPeakApproach(
			t,
			c.config.DynamicPeakApproaches,
			c.bessSoe.value,
			c.config.BessChargeEfficiency,
			c.config.ModoClient,
		),
		basicImportAvoidance(
			t,
			c.config.ImportAvoidancePeriods,
			c.SitePower(),
			c.lastBessTargetPower,
		),
		basicExportAvoidance(
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

	action := c.prioritiseControlComponents(components)

	slog.Info(
		"Controlling BESS",
		"site_power", c.sitePower.value,
		"bess_soe", c.bessSoe.value,
		"control_components_active", action.activeComponentNames,
		"constraint_site_power_active", action.constraints.sitePower,
		"constraint_bess_power_active", action.constraints.bessPower,
		"constraint_bess_soe_active", action.constraints.bessSoe,
		"rates_import", ratesImport,
		"rates_export", ratesExport,
		"bess_last_target_power", c.lastBessTargetPower,
		"bess_target_power", action.bessTargetPower,
	)

	command := telemetry.BessCommand{
		TargetPower: action.bessTargetPower,
	}
	sendIfNonBlocking(c.config.BessCommands, command, "PowerPack commands")
	c.lastBessTargetPower = action.bessTargetPower
}

// prioritisedAction just helps organise the return values of `prioritiseControlComponents`
type prioritisedAction struct {
	bessTargetPower      float64           // the power that the bess should deliver
	constraints          activeConstraints // any constraints that were used when calculating the `bessTargetPower` (useful for logging)
	activeComponentNames string            // comma-separated names of any components that were used to calculate the `bessTargetPower` (useful for logging)
}

// prioritiseControlComponents runs through all the given components and decides the appropriate action to take.
// Some components don't allow any lower-priority components to change anything, whereas others will allow lower-priority
// components to adjust the power level - this is controlled by min and max power limits on each component.
func (c *Controller) prioritiseControlComponents(components []controlComponent) prioritisedAction {

	// These vars keep track of the current power levels as they are altered by the various components that we see.
	// They are nil to start with to indicate that there has been no value set.
	var power *float64
	var minPower *float64
	var maxPower *float64

	activeComponentNames := "" // Keep track of the names of any active components for debug logging

	for _, component := range components {

		if component.isActive() {
			activeComponentNames = fmt.Sprintf("%s,%s", activeComponentNames, component.name)
		} else {
			continue // nothing to do with this component, move onto the next one
		}

		// slog.Info("Processing control component", "name", component.name, "target_power", strForPointerToFloat64(component.targetPower), "min_target_power", strForPointerToFloat64(component.minTargetPower), "max_target_power", strForPointerToFloat64(component.minTargetPower))

		// If this component specifies a target power then update the `power` if it's within any existing bounds
		if component.targetPower != nil {
			if ((minPower == nil) || (*component.targetPower >= *minPower)) &&
				((maxPower == nil) || (*component.targetPower <= *maxPower)) {
				power = component.targetPower
			}
		}

		// If this component specifies a minimum target power limit then update the `minPower` if it's within any existing bounds
		if component.minTargetPower != nil {
			if (minPower == nil) || (*component.minTargetPower > *minPower) {
				// this component specifies a minTargetPower that is higher than any previous limit
				if (power == nil) || (*component.minTargetPower <= *power) {
					minPower = component.minTargetPower
				} else {
					slog.Error("Component's min target power puts the current power out of bounds - ignoring", "target_power", *power, "component_min_power", *component.minTargetPower, "component_name", component.name)
				}
			}
		}

		// If this component specifies a maximum target power limit then update the `maxPower` if it's within any existing bounds
		if component.maxTargetPower != nil {
			if (maxPower == nil) || (*component.maxTargetPower < *maxPower) {
				// this component specifies a maxTargetPower that is lower than any previous limit
				if (power == nil) || (*component.maxTargetPower >= *power) {
					maxPower = component.maxTargetPower
				} else {
					slog.Error("Component's max target power puts the current power out of bounds - ignoring", "target_power", *power, "component_min_power", *component.maxTargetPower, "component_name", component.name)
				}
			}
		}
	}

	if power == nil {
		return prioritisedAction{
			bessTargetPower:      0.0,
			constraints:          activeConstraints{},
			activeComponentNames: "idle",
		}
	}

	constrainedPower, activeConstraints := c.constrainedBessPower(*power)

	return prioritisedAction{
		bessTargetPower:      constrainedPower,
		constraints:          activeConstraints,
		activeComponentNames: activeComponentNames,
	}
}

// constrainedBessPower returns the power level that should be sent to the BESS, after taking account of BESS inverter and site grid connection constraints.
// Limits are applied to keep the SoE, BESS power, and site power within bounds. Details of which limits were activated in the calculation are returned.
func (c *Controller) constrainedBessPower(rawTargetPower float64) (float64, activeConstraints) {

	var bessPowerLimitsActive1 bool
	var sitePowerLimitsActive bool
	var bessSoeLimitActive bool

	// Apply the physical power limits of the BESS inverter
	constrainedTargetPower, bessPowerLimitsActive1 := limitValue(rawTargetPower, c.config.BessDischargePowerLimit, c.config.BessChargePowerLimit)

	// The target power defines the power level at the BESS inverter, but we must ensure that we don't exceed the site connection limits.
	bessPowerDiff := constrainedTargetPower - c.lastBessTargetPower
	expectedSitePower := c.SitePower() - bessPowerDiff // Site power: positive is import, negative is export. Battery power: positive is discharge, negative is charge.
	if expectedSitePower > c.config.SiteImportPowerLimit {
		// We would be exeeding the import limit - so instead set the target power so that it hits the import limit
		err := c.config.SiteImportPowerLimit - c.SitePower()
		constrainedTargetPower = c.lastBessTargetPower - err
		sitePowerLimitsActive = true
	} else if expectedSitePower < -c.config.SiteExportPowerLimit {
		// We would be exeeding the export limit - so instead set the target power so that it hits the export limit
		err := -c.config.SiteExportPowerLimit - c.SitePower()
		constrainedTargetPower = c.lastBessTargetPower - err
		sitePowerLimitsActive = true
	}

	// TODO: there are some edge-case scenarios where the sign of the target power could change - e.g. if solar exports exceed the site limits.
	// In that scenario we might just want to turn the battery off?

	// Apply BESS SoE limits
	if constrainedTargetPower > 0 && c.bessSoe.value <= c.config.BessSoeMin {
		constrainedTargetPower = 0
		bessSoeLimitActive = true
	}
	if constrainedTargetPower < 0 && c.bessSoe.value >= c.config.BessSoeMax {
		constrainedTargetPower = 0
		bessSoeLimitActive = true
	}

	return constrainedTargetPower, activeConstraints{
		bessPower: bessPowerLimitsActive1,
		sitePower: sitePowerLimitsActive,
		bessSoe:   bessSoeLimitActive,
	}
}

// maxBessDischarge returns the maximum discharge rate of the BESS at this point in time.
func (c *Controller) maxBessDischarge() float64 {
	// Use the existing `constrainedBessPower` method to apply limits onto an infinite requested power.
	maxBessDischarge, _ := c.constrainedBessPower(math.Inf(+1))
	return maxBessDischarge
}
