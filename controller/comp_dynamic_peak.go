package controller

import (
	"math"
	"time"

	"github.com/cepro/besscontroller/cartesian"
	"github.com/cepro/besscontroller/config"
	timeutils "github.com/cepro/besscontroller/time_utils"
	"golang.org/x/exp/slog"
)

// dynamicPeakDischarge returns the control component for discharging the battery into a peak - usually associated with a DUoS red band - preferring to discharge into short periods and microgrid loads.
func dynamicPeakDischarge(t time.Time, configs []config.DynamicPeakDischargeConfig, bessSoe, sitePower, lastTargetPower, maxBessDischarge float64, modoClient imbalancePricer) controlComponent {

	logger := slog.Default()

	// First find any dynamic peak conigurations that are "in the peak" at the moment
	conf, absPeriod := findPeriodicalConfigForTime(t, configs)
	if conf == nil {
		return INACTIVE_CONTROL_COMPONENT
	}

	peakEnd := absPeriod.End

	controlComponentName := "dynamic_peak_discharge"

	maxDischargeComponent := controlComponent{
		name:         controlComponentName,
		status:       componentStatusActiveGreedy,
		targetPower:  math.Inf(1), // discharge as fast possible - this will be capped by the various downstream checks
		controlPoint: controlPointBess,
	}

	// availableEnergy is how much energy we have to discharge before we reach the target
	availableEnergy := bessSoe - conf.TargetSoe
	if availableEnergy <= 0 {
		logger.Info("Dynamic peak doesn't have enough energy", "available_energy", availableEnergy)
		return INACTIVE_CONTROL_COMPONENT
	}

	// assumedDurationToEmpty is an approximation because we may be limited by grid constraints which depend on the load and solar levels
	// which, in turn, may change throughout the peak
	assumedDurationToEmpty := time.Duration((availableEnergy / maxBessDischarge) * float64(time.Hour))

	// We want to ensure that we empty the battery completely by the end of the peak period, and there is a
	// point into the peak where we must discharge at the max power to ensure that.
	latestTimeBeforeMaxDischarge := peakEnd.Add(-assumedDurationToEmpty)
	if t.After(latestTimeBeforeMaxDischarge) {
		logger.Info("Dynamic peak discharging at max because there is excess energy", "latest_time_before_max_discharge", latestTimeBeforeMaxDischarge)
		return maxDischargeComponent
	}

	// We are early enough in the peak period to have some flexibility about how much we discharge, use the imbalance prediction
	// to inform how hard we discharge now.
	_, imbalanceVolume, gotPrediction := predictImbalance(
		t,
		config.NivPredictionConfig{
			WhenShort: conf.ShortPrediction,
			// We are only really interested in predicting a short scenario, so don't allow predictions for long
			WhenLong: config.NivPredictionDirectionConfig{AllowPrediction: false},
		},
		modoClient,
	)

	if !gotPrediction || imbalanceVolume < 0 {
		// either we don't know what the system state is, or the system is long (relatively low prices)
		if conf.PrioritiseResidualLoad {
			// Even though the system is long, discharge to avoid microgrid imports (if any)
			logger.Info("Dynamic peak doing import avoidance to wait for short system", "got_prediction", gotPrediction, "imbalance_volume", imbalanceVolume, "latest_time_before_max_discharge", latestTimeBeforeMaxDischarge)
			return importAvoidanceHelper(sitePower, lastTargetPower, controlComponentName, false)
		}
		// If we are not 'prioritising loads' then hold off on the discharge completely until the last minute,
		// or until the system is short.
		logger.Info("Dynamic peak doing nothing to wait for short system", "got_prediction", gotPrediction, "imbalance_volume", imbalanceVolume)
		return INACTIVE_CONTROL_COMPONENT
	}

	// System is short

	if !conf.PrioritiseResidualLoad {
		// If we are not 'prioritising loads' then just discharge at the max power when the system is short
		logger.Info("Dynamic peak discharging at max due to short system")
		return maxDischargeComponent
	}

	// Here we want to discharge at the max power we can, whilst ensuring there is enough energy
	// in the battery to service any residual microgrid load at the end of the peak.
	// Approximate that the residual load (i.e. microgrid load minus microgrid solar generation) will
	// stay the same throughout the peak.
	// TODO: we could make some assumption about the residual growing due to less solar later on?
	microgridResidualPower := sitePower + lastTargetPower // infer the microgrid load from the site meter and the last bess power
	if microgridResidualPower <= 0 {
		// There is no residual load (probably due to solar excess) so just discharge at max power
		logger.Info("Dynamic peak discharging at max due to short system and no residual power", "microgrid_residual_power", microgridResidualPower)
		return maxDischargeComponent
	}
	durationToEndOfPeak := peakEnd.Sub(t)
	reserveEnergy := microgridResidualPower * durationToEndOfPeak.Hours()

	// If we have more energy than we need to keep in reserve then discharge as hard as we can (system is short so prices are currently good)
	// until we run out of excess energy and can only meet the reserve requirement - at that point do import avoidance.
	// TODO: We can do slightly better than this logic because we get more certain about the imbalance direction later in the settlement period,
	//       so, if we know we are going to drain the battery, it would be better to do it in the last 10 mins of the SP than the first 10mins.
	//       However, the impact on revenue is probably quite small.
	if availableEnergy > reserveEnergy {
		logger.Info("Dynamic peak discharging at max due to short system and more energy than reserve", "available_energy", availableEnergy, "reserve_energy", reserveEnergy)
		return maxDischargeComponent
	} else {
		logger.Info("Dynamic peak doing import avoidance due to short system and less energy than reserve", "available_energy", availableEnergy, "reserve_energy", reserveEnergy)
		return importAvoidanceHelper(sitePower, lastTargetPower, controlComponentName, false)
	}
}

// dynamicPeakApproach returns the control component associated with approaching a peak
func dynamicPeakApproach(t time.Time, configs []config.DynamicPeakApproachConfig, bessSoe, chargeEfficiency float64, modoClient imbalancePricer) controlComponent {

	controlComponentName := "dynamic_peak_approach"
	logger := slog.Default()

	for _, conf := range configs {

		if !conf.PeakPeriod.Days.IsOnDay(t) {
			// This won't work if the approach curve crosses over a midnight boundary
			continue
		}

		peakPeriod := conf.PeakPeriod.ClockTimePeriod.AbsolutePeriodOnDate(t.Year(), t.Month(), t.Day())

		referencePoint := datetimePoint(t, bessSoe)
		hoursLeftOfSP := float64(timeutils.DurationLeftOfSP(t)) / float64(time.Hour)

		// First check if there is a requirement to "encourage charge" if the system is long
		_, imbalanceVolume, gotPrediction := predictImbalance(
			t,
			config.NivPredictionConfig{
				// We are only really interested in predicting a long scenario, so don't allow predictions for short
				WhenShort: config.NivPredictionDirectionConfig{AllowPrediction: false},
				WhenLong:  conf.LongPrediction,
			},
			modoClient,
		)
		if gotPrediction && imbalanceVolume < 0 {
			// system is long

			// use the 'encourage to soe' value if specified, but fall back to the 'to soe' value if it's not present
			toSoe := conf.EncourageToSoe
			if toSoe == 0.0 {
				toSoe = conf.ToSoe
			}

			encourageCurve := approachCurve(
				peakPeriod,
				toSoe,
				chargeEfficiency,
				conf.AssumedChargePower,
				conf.EncourageChargeDurationFactor,
				time.Duration(float64(time.Minute)*conf.ChargeCushionMins),
			)
			encourageEnergy := encourageCurve.VerticalDistance(referencePoint)
			encouragePower := (encourageEnergy / hoursLeftOfSP) / chargeEfficiency

			logger.Info(
				"Dynamic approach debug",
				"encourage_energy", encourageEnergy,
				"encourage_power", encouragePower,
				"peak_start", peakPeriod.Start,
				"to_soe", toSoe,
			)

			if !math.IsNaN(encouragePower) && encouragePower > 0 {
				return controlComponent{
					name:         controlComponentName,
					status:       componentStatusActiveAllowMoreCharge,
					targetPower:  -encouragePower,
					controlPoint: controlPointBess,
				}
			}
		}

		// There wasn't a requirement to "encourage charge", but we may need to "force charge"
		forceCurve := approachCurve(
			peakPeriod,
			conf.ToSoe,
			chargeEfficiency,
			conf.AssumedChargePower,
			conf.ForceChargeDurationFactor,
			time.Duration(float64(time.Minute)*conf.ChargeCushionMins),
		)
		forceEnergy := forceCurve.VerticalDistance(referencePoint)
		forcePower := (forceEnergy / hoursLeftOfSP) / chargeEfficiency

		if !math.IsNaN(forcePower) && forcePower > 0 {
			return controlComponent{
				name:         controlComponentName,
				status:       componentStatusActiveAllowMoreCharge,
				targetPower:  -forcePower,
				controlPoint: controlPointBess,
			}
		}
	}

	return INACTIVE_CONTROL_COMPONENT
}

// approachCurve returns a curve representing the boundary of the peak approach
func approachCurve(peakPeriod timeutils.Period, toSoe, chargeEfficiency, assumedChargePower, chargeDurationFactor float64, chargeCushion time.Duration) cartesian.Curve {

	// how long is the approach
	approachHours := ((toSoe / assumedChargePower) / chargeEfficiency) * chargeDurationFactor
	approachDuration := time.Duration(float64(time.Hour) * approachHours)

	approachCurve := cartesian.Curve{
		Points: []cartesian.Point{
			datetimePoint(peakPeriod.Start.Add(-approachDuration-chargeCushion), 0),
			datetimePoint(peakPeriod.Start.Add(-chargeCushion), toSoe),
			datetimePoint(peakPeriod.End, toSoe),
		},
	}

	return approachCurve
}

func datetimePoint(t time.Time, y float64) cartesian.Point {
	// Returns a Point object that encodes a time of day.
	// This uses a reference datetime to convert a time into a float number of seconds, so may not work over midnight
	// boundaries.
	referenceTime := time.Date(2000, 1, 1, 0, 0, 0, 0, t.Location())
	duration := t.Sub(referenceTime) / time.Second // integer truncation of number of seconds isn't significant for our use cases
	return cartesian.Point{
		X: float64(duration),
		Y: y,
	}
}
