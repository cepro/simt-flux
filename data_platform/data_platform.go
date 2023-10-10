package dataplatform

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"time"

	"github.com/cepro/besscontroller/repository"
	"github.com/cepro/besscontroller/telemetry"
	"github.com/google/uuid"

	supa "github.com/nedpals/supabase-go"
)

// DataPlatform handles the streaming of telemetry to Supabase.
// Put new meter and bess readings onto the appropriate channels, they will be bufferred on disk in a SQLite database before
// being uploaded to Supabase.
type DataPlatform struct {
	BessReadings  chan telemetry.BessReading
	MeterReadings chan telemetry.MeterReading

	// these maps hold the last reading received, keyed by the device ID
	latestBessReadings  map[uuid.UUID]telemetry.BessReading
	latestMeterReadings map[uuid.UUID]telemetry.MeterReading

	repository *repository.Repository
	supaClient *supa.Client
}

func New(supabaseUrl string, supabaseAnonKey string, supabaseUserKey string, schema string, bufferRepositoryFilename string) (*DataPlatform, error) {

	supaClient := supa.CreateClient(supabaseUrl, supabaseAnonKey)

	// The supabase client library doesn't have a fully featured interface, here we specify options directly by
	// adding headers to the postgrest requests.
	// Use the appropriate schema:
	supaClient.DB.AddHeader("Accept-Profile", schema)
	supaClient.DB.AddHeader("Content-Profile", schema)
	// Use a user JWT:
	if supabaseUserKey != "" {
		supaClient.DB.AddHeader("Authorization", fmt.Sprintf("Bearer %s", supabaseUserKey))
	}

	repository, err := repository.New(bufferRepositoryFilename)
	if err != nil {
		return nil, fmt.Errorf("create repository: %w", err)
	}

	return &DataPlatform{
		BessReadings:        make(chan telemetry.BessReading, 25), // a small buffer to allow SQLite to catch up in case the disk is slow
		MeterReadings:       make(chan telemetry.MeterReading, 25),
		latestBessReadings:  make(map[uuid.UUID]telemetry.BessReading),
		latestMeterReadings: make(map[uuid.UUID]telemetry.MeterReading),
		repository:          repository,
		supaClient:          supaClient,
	}, nil
}

// Run loops forever waiting for meter or bess readings, when they are available they are uploaded.
func (d *DataPlatform) Run(ctx context.Context, uploadInterval time.Duration) {

	// TODO: would be nice if this was "on the minute"
	uploadTicker := time.NewTicker(uploadInterval)

	for {
		select {
		case <-ctx.Done():
			return
		case reading := <-d.BessReadings:
			d.latestBessReadings[reading.DeviceID] = reading

		case reading := <-d.MeterReadings:
			d.latestMeterReadings[reading.DeviceID] = reading

		case _ = <-uploadTicker.C:
			d.runUpload()

		}
	}
}

func (d *DataPlatform) runUpload() {

	slog.Debug("Running data platform upload...")

	for _, reading := range d.latestBessReadings {
		err := d.repository.AddBessReading(reading)
		if err != nil {
			slog.Error("failed to persist bess reading", "error", err)
		}
	}
	d.latestBessReadings = make(map[uuid.UUID]telemetry.BessReading)

	for _, reading := range d.latestMeterReadings {
		err := d.repository.AddMeterReading(reading)
		if err != nil {
			slog.Error("failed to persist meter reading", "error", err)
		}
	}
	d.latestMeterReadings = make(map[uuid.UUID]telemetry.MeterReading)

	// TODO: we could attempt the first upload before storing to disk, and only store to disk if the upload fails
	d.uploadRepository()
}

// uploadRepository attempts to upload the telemetry from the repository into Supabase.
func (d *DataPlatform) uploadRepository() {

	// uploadChunkLimit defines how many data points we can upload in one supabase HTTP request
	uploadChunkLimit := 100

	// first attempt to upload any new readings that have not been seen before
	freshBessReadings, err := d.repository.GetBessReadings(uploadChunkLimit, true)
	if err != nil {
		slog.Error("failed to query fresh bess readings", "error", err)
	} else if len(freshBessReadings) > 1 {
		err = d.handleReadings(freshBessReadings)
		if err != nil {
			slog.Error("failed to handle fresh bess readings", "error", err)
		}
	}
	freshMeterReadings, err := d.repository.GetMeterReadings(uploadChunkLimit, true)
	if err != nil {
		slog.Error("failed to query fresh meter readings", "error", err)
	} else if len(freshMeterReadings) > 1 {
		err = d.handleReadings(freshMeterReadings)
		if err != nil {
			slog.Error("failed to handle fresh meter readings", "error", err)
		}
	}

	// then attempt to upload any old readings that have already failed an upload at least once
	oldBessReadings, err := d.repository.GetBessReadings(uploadChunkLimit, false)
	if err != nil {
		slog.Error("failed to query old meter readings", "error", err)
	} else if len(oldBessReadings) > 1 {
		err = d.handleReadings(oldBessReadings)
		if err != nil {
			slog.Error("failed to handle old meter readings", "error", err)
		}
	}
	oldMeterReadings, err := d.repository.GetMeterReadings(uploadChunkLimit, false)
	if err != nil {
		slog.Error("failed to query old meter readings", "error", err)
	} else if len(oldMeterReadings) > 1 {
		err = d.handleReadings(oldMeterReadings)
		if err != nil {
			slog.Error("failed to handle old meter readings", "error", err)
		}
	}

	return
}

// handleReadings attempts to upload the given readings. If successfull, it deletes the readings from the database, if
// unsuccessful, it increments the 'upload attempt count' column and leaves the reading in the database for another time.
func (d *DataPlatform) handleReadings(readings interface{}) error {

	convertedReadings, tableName := getReadingsForSupabase(readings)
	// TODO: organise error better
	uploadErr := d.supaClient.DB.From(tableName).Insert(convertedReadings).Execute(nil)
	if uploadErr != nil {
		uploadErr := fmt.Errorf("upload failed: %w", uploadErr)
		errInc := d.repository.IncrementUploadAttemptCount(readings)
		if errInc != nil {
			return fmt.Errorf("%w: increment upload attempt count: %w", uploadErr, errInc)
		}
		return uploadErr
	}

	deleteErr := d.repository.DeleteReadings(readings)
	if deleteErr != nil {
		return fmt.Errorf("delete meter readings (%+v): %w", readings, deleteErr)
	}

	slog.Info("Uploaded readings", "db_table", tableName, "db_records", reflect.ValueOf(readings).Len())

	// TODO: really think through this logic to handle edge cases, e.g. where the upload succeeds but the delete doesn't

	return nil
}
