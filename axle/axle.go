package axle

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/cepro/besscontroller/telemetry"
	"github.com/google/uuid"
)

// Axle handles the Axle API. We send them operational telemetry and they give us control schedules.
// At the moment schedules are retrieved via polling initiated here.
type Axle struct {
	BessReadings  chan telemetry.BessReading  // put new bess readings here and they will be uploaded as telemetry to Axle
	MeterReadings chan telemetry.MeterReading // put new meter readings here and they will be uploaded as telemetry to Axle

	schedules chan<- Schedule // new schedules will be placed onto this channel as they are received

	siteMeterID uuid.UUID
	bessMeterID uuid.UUID

	host   string
	logger *slog.Logger

	// these maps hold the last reading received, keyed by the device ID
	latestBessReadings  map[uuid.UUID]telemetry.BessReading
	latestMeterReadings map[uuid.UUID]telemetry.MeterReading

	latestSchedule Schedule
}

func New(schedules chan<- Schedule, host string, siteMeterID, bessMeterID uuid.UUID) *Axle {

	logger := slog.Default().With("host", host)

	return &Axle{
		BessReadings:        make(chan telemetry.BessReading, 25), // TODO: check this size. A small buffer to allow things to catch up in case the upload is slow
		MeterReadings:       make(chan telemetry.MeterReading, 25),
		schedules:           schedules,
		siteMeterID:         siteMeterID,
		bessMeterID:         bessMeterID,
		host:                host,
		logger:              logger,
		latestBessReadings:  make(map[uuid.UUID]telemetry.BessReading),
		latestMeterReadings: make(map[uuid.UUID]telemetry.MeterReading),
	}
}

// Run loops forever and manages the API. Exits when the context is cancelled.
func (a *Axle) Run(ctx context.Context, telemetryUploadInterval, schedulePullInterval time.Duration) error {

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
func (a *Axle) uploadOperationalTelemetry() {

	var err error

	// TODO: implement, we probably want to map between internal meter UUIDs and "battery inverter"/blah/blah etc
	// bessMeterReading, ok := a.latestMeterReadings[a.bessMeterID]
	// siteMeterReading, ok := a.latestMeterReadings[a.siteMeterID]

	if err != nil {
		a.logger.Info("Failed Axle operational telemetry upload", "error", err)
	} else {
		slog.Info("Axle operational telemetry uploaded", "bess_readings", len(a.latestBessReadings), "meter_readings", len(a.latestMeterReadings))
	}
}

// processSchedule polls the latest schedule from Axle and forwards it down the channel
func (a *Axle) processSchedule() {

	now := time.Now()

	scheduleItems := make([]ScheduleItem, 1)
	json.Unmarshal([]byte(hardCodedResponse), &scheduleItems)

	schedule := Schedule{
		ReceivedTime: now,
		Items:        scheduleItems,
	} // TODO: query for new schedule from Axle API

	if !a.latestSchedule.Equal(schedule, false) {
		a.logger.Info("Pulled new schedule from Axle", "schedule", schedule)
		a.latestSchedule = schedule
		a.schedules <- schedule
	} else {
		a.logger.Info("Pulled schedule from Axle, but it hasn't changed")
	}

}

var hardCodedResponse = `
[
	{
        "start_timestamp": "2025-04-29T20:00:00+01:00",
        "end_timestamp": "2025-04-29T20:05:00+01:00",
        "action": "charge_max",
        "allow_deviation": false
    },
	{
        "start_timestamp": "2025-04-29T20:05:00+01:00",
        "end_timestamp": "2025-04-29T20:10:00+01:00",
        "action": "avoid_import",
        "allow_deviation": false
    },
    {
        "start_timestamp": "2025-04-30T00:00:00+01:00",
        "end_timestamp": "2025-04-30T03:00:00+01:00",
        "action": "charge_max",
        "allow_deviation": false
    },
    {
        "start_timestamp": "2025-05-01T00:00:00+01:00",
        "end_timestamp": "2025-05-01T03:00:00+01:00",
        "action": "charge_max",
        "allow_deviation": false
    },
    {
        "start_timestamp": "2025-05-02T00:00:00+01:00",
        "end_timestamp": "2025-05-02T03:00:00+01:00",
        "action": "charge_max",
        "allow_deviation": false
    },
    {
        "start_timestamp": "2025-05-03T00:00:00+01:00",
        "end_timestamp": "2025-05-03T03:00:00+01:00",
        "action": "charge_max",
        "allow_deviation": false
    },
    {
        "start_timestamp": "2025-05-04T00:00:00+01:00",
        "end_timestamp": "2025-05-04T03:00:00+01:00",
        "action": "charge_max",
        "allow_deviation": false
    },
    {
        "start_timestamp": "2025-05-05T00:00:00+01:00",
        "end_timestamp": "2025-05-05T03:00:00+01:00",
        "action": "charge_max",
        "allow_deviation": false
    },
    {
        "start_timestamp": "2025-05-06T00:00:00+01:00",
        "end_timestamp": "2025-05-06T03:00:00+01:00",
        "action": "charge_max",
        "allow_deviation": false
    },
    {
        "start_timestamp": "2025-05-07T00:00:00+01:00",
        "end_timestamp": "2025-05-07T03:00:00+01:00",
        "action": "charge_max",
        "allow_deviation": false
    },
    {
        "start_timestamp": "2025-05-08T00:00:00+01:00",
        "end_timestamp": "2025-05-08T03:00:00+01:00",
        "action": "avoid_import",
        "allow_deviation": true
    },
    {
        "start_timestamp": "2025-05-09T00:00:00+01:00",
        "end_timestamp": "2025-05-09T03:00:00+01:00",
        "action": "avoid_import",
        "allow_deviation": true
    },
    {
        "start_timestamp": "2025-05-10T00:00:00+01:00",
        "end_timestamp": "2025-05-10T03:00:00+01:00",
        "action": "charge_max",
        "allow_deviation": false
    },
    {
        "start_timestamp": "2025-05-11T00:00:00+01:00",
        "end_timestamp": "2025-05-11T03:00:00+01:00",
        "action": "avoid_import",
        "allow_deviation": true
    },
    {
        "start_timestamp": "2025-05-12T00:00:00+01:00",
        "end_timestamp": "2025-05-12T03:00:00+01:00",
        "action": "avoid_import",
        "allow_deviation": true
    },
    {
        "start_timestamp": "2025-05-13T00:00:00+01:00",
        "end_timestamp": "2025-05-13T03:00:00+01:00",
        "action": "avoid_import",
        "allow_deviation": true
    },
    {
        "start_timestamp": "2025-05-14T00:00:00+01:00",
        "end_timestamp": "2025-05-14T03:00:00+01:00",
        "action": "avoid_import",
        "allow_deviation": true
    },
    {
        "start_timestamp": "2025-05-15T00:00:00+01:00",
        "end_timestamp": "2025-05-15T03:00:00+01:00",
        "action": "avoid_import",
        "allow_deviation": true
    },
    {
        "start_timestamp": "2025-05-16T00:00:00+01:00",
        "end_timestamp": "2025-05-16T03:00:00+01:00",
        "action": "avoid_import",
        "allow_deviation": true
    },
    {
        "start_timestamp": "2025-05-17T00:00:00+01:00",
        "end_timestamp": "2025-05-17T03:00:00+01:00",
        "action": "avoid_import",
        "allow_deviation": true
    },
    {
        "start_timestamp": "2025-05-18T00:00:00+01:00",
        "end_timestamp": "2025-05-18T03:00:00+01:00",
        "action": "avoid_import",
        "allow_deviation": true
    },
    {
        "start_timestamp": "2025-05-19T00:00:00+01:00",
        "end_timestamp": "2025-05-19T03:00:00+01:00",
        "action": "avoid_import",
        "allow_deviation": true
    },
    {
        "start_timestamp": "2025-05-20T00:00:00+01:00",
        "end_timestamp": "2025-05-20T03:00:00+01:00",
        "action": "avoid_import",
        "allow_deviation": true
    },
    {
        "start_timestamp": "2025-05-21T00:00:00+01:00",
        "end_timestamp": "2025-05-21T03:00:00+01:00",
        "action": "avoid_import",
        "allow_deviation": true
    },
    {
        "start_timestamp": "2025-05-22T00:00:00+01:00",
        "end_timestamp": "2025-05-22T03:00:00+01:00",
        "action": "avoid_import",
        "allow_deviation": true
    },
    {
        "start_timestamp": "2025-05-23T00:00:00+01:00",
        "end_timestamp": "2025-05-23T03:00:00+01:00",
        "action": "avoid_import",
        "allow_deviation": true
    },
    {
        "start_timestamp": "2025-05-24T00:00:00+01:00",
        "end_timestamp": "2025-05-24T03:00:00+01:00",
        "action": "avoid_import",
        "allow_deviation": true
    },
    {
        "start_timestamp": "2025-05-25T00:00:00+01:00",
        "end_timestamp": "2025-05-25T03:00:00+01:00",
        "action": "avoid_import",
        "allow_deviation": true
    },
    {
        "start_timestamp": "2025-05-26T00:00:00+01:00",
        "end_timestamp": "2025-05-26T03:00:00+01:00",
        "action": "avoid_import",
        "allow_deviation": true
    }
]`
