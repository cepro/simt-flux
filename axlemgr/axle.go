package axlemgr

import (
	"context"
	"log/slog"
	"time"

	"github.com/cepro/besscontroller/axleclient"
	"github.com/cepro/besscontroller/telemetry"
	"github.com/google/uuid"
)

// AxleMgr controls the flow of information to and from Axle. We send them operational telemetry and they give us control schedules.
// At the moment schedules are retrieved via polling initiated here.
type AxleMgr struct {
	BessReadings  chan telemetry.BessReading  // put new bess readings here and the relevant data will be uploaded to Axle
	MeterReadings chan telemetry.MeterReading // put new meter readings here and the relevant data will be uploaded to Axle

	schedules chan<- axleclient.Schedule // new schedules will be placed onto this channel as they are received

	axleAssetID string    // the ID that axle uses to identify this asset
	bessID      uuid.UUID // these are our IDs for the BESS and relevant meters
	siteMeterID uuid.UUID
	bessMeterID uuid.UUID

	bessNameplateEnergy float64 // this is required to convert the SoE kWh to a percentage (Axle API wants a percentage)

	client *axleclient.Client // The underlying API client to use to communicate with Axle
	logger *slog.Logger

	// these maps hold the last reading received on the channels, keyed by the device ID
	latestBessReadings  map[uuid.UUID]telemetry.BessReading
	latestMeterReadings map[uuid.UUID]telemetry.MeterReading

	latestSchedule axleclient.Schedule
}

func New(schedules chan<- axleclient.Schedule, client *axleclient.Client, bessID, siteMeterID, bessMeterID uuid.UUID, axleAssetID string, bessNameplateEnergy float64) *AxleMgr {

	return &AxleMgr{
		BessReadings:        make(chan telemetry.BessReading, 25), // A small buffer to allow things to catch up in case the upload is slow
		MeterReadings:       make(chan telemetry.MeterReading, 25),
		schedules:           schedules,
		axleAssetID:         axleAssetID,
		bessID:              bessID,
		siteMeterID:         siteMeterID,
		bessMeterID:         bessMeterID,
		bessNameplateEnergy: bessNameplateEnergy,
		client:              client,
		logger:              slog.Default(),
		latestBessReadings:  make(map[uuid.UUID]telemetry.BessReading),
		latestMeterReadings: make(map[uuid.UUID]telemetry.MeterReading),
	}
}

// Run loops forever and manages the API. Exits when the context is cancelled.
func (a *AxleMgr) Run(ctx context.Context, telemetryUploadInterval, schedulePullInterval time.Duration) error {

	uploadTicker := time.NewTicker(telemetryUploadInterval)
	schedulePullTicker := time.NewTicker(schedulePullInterval)

	a.logger.Info("Starting axle API", "telemetry_upload_interval", telemetryUploadInterval, "schedule_poll_interval", schedulePullInterval)

	// pull the schedule from Axle immediately (don't wait for the `schedulePullInterval`)
	a.processSchedule()

	for {
		select {
		case <-ctx.Done():
			return nil
		case reading := <-a.BessReadings:
			a.latestBessReadings[reading.DeviceID] = reading

		case reading := <-a.MeterReadings:
			a.latestMeterReadings[reading.DeviceID] = reading

		case <-uploadTicker.C:
			a.uploadOperationalTelemetry()

		case <-schedulePullTicker.C:
			a.processSchedule()

		}
	}
}

// uploadOperationalTelemetry sends any operational telemetry we have to Axle
func (a *AxleMgr) uploadOperationalTelemetry() {

	var err error

	var bessReading *telemetry.BessReading
	var bessMeterReading *telemetry.MeterReading
	var siteMeterReading *telemetry.MeterReading

	if reading, ok := a.latestBessReadings[a.bessID]; ok {
		bessReading = &reading
	}

	if reading, ok := a.latestMeterReadings[a.bessMeterID]; ok {
		bessMeterReading = &reading
	}

	if reading, ok := a.latestMeterReadings[a.siteMeterID]; ok {
		siteMeterReading = &reading
	}

	axleReadings := a.getAxleReadings(bessReading, bessMeterReading, siteMeterReading)

	numReadings := len(axleReadings)
	if numReadings < 1 {
		a.logger.Warn("No readings to send to Axle")
		return
	}

	a.client.UploadReadings(axleReadings)

	if err != nil {
		a.logger.Info("Failed Axle operational telemetry upload", "error", err)
	} else {
		a.logger.Info("Axle operational telemetry uploaded", "num_readings", len(axleReadings))
	}
}

// processSchedule polls the latest schedule from Axle and forwards it down the channel
func (a *AxleMgr) processSchedule() {

	schedule, err := a.client.GetSchedule(a.axleAssetID)
	if err != nil {
		a.logger.Error("Failed to pull latest schedule", "error", err)
		return
	}

	if !a.latestSchedule.Equal(schedule, false) {
		a.logger.Info("Pulled new schedule from Axle", "schedule", schedule)
	} else {
		a.logger.Info("Pulled schedule from Axle, but it hasn't changed")
	}
	// No harm in sending the schedule even if it hasn't changed - if the reciever wants to check to for changes they can
	a.latestSchedule = schedule
	a.schedules <- schedule

}

// getAxleReadings converts the given telemetry.BessReading and telemetry.MeterReading to axleclient.Reading instances.
// Axle has it's own categorisation and structure for storing readings so here we just convert from our form to their form.
func (a *AxleMgr) getAxleReadings(bessReading *telemetry.BessReading, bessMeterReading, siteMeterReading *telemetry.MeterReading) []axleclient.Reading {

	readings := []axleclient.Reading{}

	if siteMeterReading != nil {
		boundary_power := siteMeterReading.PowerTotalActive
		t := siteMeterReading.Time
		if boundary_power != nil {
			if *boundary_power >= 0 {
				readings = append(readings, axleclient.Reading{
					AssetId:        a.axleAssetID,
					StartTimestamp: t,
					EndTimestamp:   t,
					Value:          *boundary_power,
					Label:          "boundary_import_kw",
				})
				readings = append(readings, axleclient.Reading{
					AssetId:        a.axleAssetID,
					StartTimestamp: t,
					EndTimestamp:   t,
					Value:          0.0,
					Label:          "boundary_export_kw",
				})
			} else {
				readings = append(readings, axleclient.Reading{
					AssetId:        a.axleAssetID,
					StartTimestamp: t,
					EndTimestamp:   t,
					Value:          0.0,
					Label:          "boundary_import_kw",
				})
				readings = append(readings, axleclient.Reading{
					AssetId:        a.axleAssetID,
					StartTimestamp: t,
					EndTimestamp:   t,
					Value:          *boundary_power * -1,
					Label:          "boundary_export_kw",
				})
			}
		}
	}

	if bessReading != nil {
		soePct := (bessReading.Soe / a.bessNameplateEnergy) * 100
		t := bessReading.Time
		readings = append(readings, axleclient.Reading{
			AssetId:        a.axleAssetID,
			StartTimestamp: t,
			EndTimestamp:   t,
			Value:          soePct,
			Label:          "battery_state_of_charge_pct",
		})
	}

	return readings
}
