package axle

import (
	"context"
	"log/slog"
	"time"

	"github.com/cepro/besscontroller/telemetry"
	timeutils "github.com/cepro/besscontroller/time_utils"
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

	schedule := Schedule{
		ReceivedTime: now,
		Actions: []ScheduleAction{
			{
				Period: timeutils.Period{
					Start: timeutils.FloorHH(now),
					End:   timeutils.FloorHH(now).Add(time.Minute * 30),
				},
				ActionType:     "discharge_max",
				AllowDeviation: false,
			},
		},
	} // TODO: query for new schedule from Axle API

	if !a.latestSchedule.Equal(schedule, false) {
		a.logger.Info("Pulled new schedule from Axle", "schedule", schedule)
		a.latestSchedule = schedule
		a.schedules <- schedule
	} else {
		a.logger.Info("Pulled schedule from Axle, but it hasn't changed", "schedule", a.latestSchedule)
	}

}
