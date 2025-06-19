package acuvim2

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/cepro/besscontroller/modbus"
	"github.com/cepro/besscontroller/telemetry"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
)

// Acuvim2Meter handles Modbus communications with the three phase Acuvim 2 meters.
// Meter readings are taken regularly and sent onto the `readings` channel.
type Acuvim2Meter struct {
	readings chan<- telemetry.MeterReading
	host     string
	id       uuid.UUID
	pt1      float64 // installed potential transformer 1 rating
	pt2      float64 // installed potential transformer 2 rating
	ct1      float64 // installed current transformer 1 rating
	ct2      float64 // installed current transformer 2 rating
	client   *modbus.Client
	logger   *slog.Logger
}

func New(readings chan<- telemetry.MeterReading, id uuid.UUID, host string, pt1 float64, pt2 float64, ct1 float64, ct2 float64) (*Acuvim2Meter, error) {

	logger := slog.Default().With("meter_id", id, "host", host)

	client, err := modbus.NewClient(host)
	if err != nil {
		return nil, fmt.Errorf("create modbus client: %w", err)
	}

	// PT and CT values could be read over modbus on startup rather then set by configuration

	return &Acuvim2Meter{
		readings: readings,
		id:       id,
		host:     host,
		pt1:      pt1,
		pt2:      pt2,
		ct1:      ct1,
		ct2:      ct2,
		client:   client,
		logger:   logger,
	}, nil
}

// Run loops forever polling telemetry from the meter every `period`. Exits when the context is cancelled.
func (a *Acuvim2Meter) Run(ctx context.Context, period time.Duration) error {

	readingTicker := time.NewTicker(period)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case t := <-readingTicker.C:

			metrics, err := a.client.PollBlocks(a, blocks)
			if err != nil {
				a.logger.Error("Failed to poll meter", "error", err)
				continue // try again next time
			}

			meterReading, err := a.metricsToMeterReading(metrics, t)
			if err != nil {
				a.logger.Error("Failed to convert metrics", "error", err)
				continue // try again next time
			}

			a.readings <- meterReading
		}
	}
}

// metricsToMeterReading converts the given map of metrics relating to a meter into a concrete `telemetry.MeterReading` instance.
func (a *Acuvim2Meter) metricsToMeterReading(metrics map[string]interface{}, t time.Time) (telemetry.MeterReading, error) {

	meterReading := telemetry.MeterReading{
		ReadingMeta: telemetry.ReadingMeta{
			ID:       uuid.New(),
			DeviceID: a.id,
			Time:     t,
		},
	}

	err := mapstructure.Decode(metrics, &meterReading)
	if err != nil {
		return telemetry.MeterReading{}, fmt.Errorf("decode metric map: %w", err)
	}

	return meterReading, nil
}
