package dataplatform

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"time"

	"github.com/cepro/besscontroller/repository"
	"github.com/cepro/besscontroller/supabase"
	"github.com/cepro/besscontroller/telemetry"
	"github.com/google/uuid"
)

const (
	maxUploadAttempts = 5
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
	supaClient *supabase.Client
}

func New(supabaseUrl string, supabaseAnonKey string, supabaseUserKey string, schema string, bufferRepositoryFilename string) (*DataPlatform, error) {

	supaClient, err := supabase.New(supabaseUrl, supabaseAnonKey, supabaseUserKey, schema)
	if err != nil {
		return nil, fmt.Errorf("create supabase client: %w", err)
	}

	repository, err := repository.New(bufferRepositoryFilename)
	if err != nil {
		return nil, fmt.Errorf("create repository: %w", err)
	}

	return &DataPlatform{
		BessReadings:        make(chan telemetry.BessReading, 25), // a small buffer to allow things to catch up in case the upload / sqlite is slow
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

			var err error
			attemptToProcessOldReadings := true
			nFreshBess := 0
			nFreshMeter := 0
			nOldBess := 0
			nOldMeter := 0

			// Process all the fresh readings. A best-effort approach is taken so that, even if there are failures, they are stored to disk
			nFreshBess, err = d.processFreshBessReadings()
			if err != nil {
				slog.Error("Failed to process fresh BESS readings", "error", err)
				attemptToProcessOldReadings = false
			}
			nFreshMeter, err = d.processFreshMeterReadings()
			if err != nil {
				slog.Error("Failed to process fresh meter readings", "error", err)
				attemptToProcessOldReadings = false
			}

			// Only attempt to re-upload old readings if the fresh readings were successfully uploaded. This approach prevents the 'upload attempt
			// count' from being incremented regularly when the network is down (if the network is down than the fresh readings would fail to upload).
			if attemptToProcessOldReadings {

				nOldBess, err = d.processOldBessReadings()
				if err != nil {
					slog.Error("Failed to process old BESS readings", "error", err)
				}

				nOldMeter, err = d.processOldMeterReadings()
				if err != nil {
					slog.Error("Failed to process old meter readings", "error", err)
				}
			}

			slog.Info("Finished supabase upload routine", "bess_readings_fresh", nFreshBess, "meter_readings_fresh", nFreshMeter, "bess_readings_old", nOldBess, "meter_readings_old", nOldMeter)
		}
	}
}

// processFreshBessReadings attempts to upload any new Bess readings
func (d *DataPlatform) processFreshBessReadings() (int, error) {
	// create an array of readings from the `latestBessReadings` map
	readings := make([]telemetry.BessReading, 0, len(d.latestBessReadings))
	for _, reading := range d.latestBessReadings {
		readings = append(readings, reading)
	}
	d.latestBessReadings = make(map[uuid.UUID]telemetry.BessReading) // start with a fresh map for future readings

	err := d.processFreshReadings(readings)
	if err != nil {
		return 0, err
	}

	return len(readings), nil
}

// processFreshMeterReadings attempts to upload any new Meter readings
func (d *DataPlatform) processFreshMeterReadings() (int, error) {
	// create an array of readings from the `latestBessReadings` map
	readings := make([]telemetry.MeterReading, 0, len(d.latestMeterReadings))
	for _, reading := range d.latestMeterReadings {
		readings = append(readings, reading)
	}
	d.latestMeterReadings = make(map[uuid.UUID]telemetry.MeterReading) // start with a fresh map for future readings

	err := d.processFreshReadings(readings)
	if err != nil {
		return 0, err
	}

	return len(readings), nil
}

// processOldBessReadings attempts to upload any stored Bess readings
func (d *DataPlatform) processOldBessReadings() (int, error) {

	// Only attempt to upload one old reading at a time, this is in case there is a 'bad apple' that is causing the batch uploads to fail
	oldBessReadings, err := d.repository.GetBessReadings(1, maxUploadAttempts)
	if err != nil {
		return 0, fmt.Errorf("retrieve bess readings: %w", err)
	}

	return d.processOldReadings(oldBessReadings)
}

// processOldMeterReadings attempts to upload any stored Meter readings
func (d *DataPlatform) processOldMeterReadings() (int, error) {

	// Only attempt to upload one old reading at a time, this is in case there is a 'bad apple' that is causing the batch uploads to fail
	oldMeterReadings, err := d.repository.GetMeterReadings(1, maxUploadAttempts)
	if err != nil {
		return 0, fmt.Errorf("retrieve meter readings: %w", err)
	}

	return d.processOldReadings(oldMeterReadings)
}

// processFreshReadings attempts to upload the given new readings, which can be of any type.
// If upload fails, then the readings will be stored in an on-disk repository until they can be uploaded.
func (d *DataPlatform) processFreshReadings(readings interface{}) error {
	uploadErr := d.supaClient.UploadReadings(readings)
	if uploadErr != nil {
		uploadErr := fmt.Errorf("upload failed: %w", uploadErr)
		storeErr := d.repository.StoreReadings(readings)
		if storeErr != nil {
			return fmt.Errorf("%w: store readings for later upload failed: %w", uploadErr, storeErr)
		}
		return uploadErr
	}
	return nil
}

// processOldReadings attempts to re-upload any of the given old/stored readings that have already failed an upload at least once.
// On success, the stored readings are deleted from the on-disk repository. On failure, the 'upload attempt count' is incremented.
func (d *DataPlatform) processOldReadings(storedReadings interface{}) (int, error) {

	len := reflect.ValueOf(storedReadings).Len()
	if len < 1 {
		return 0, nil
	}

	// pull out the 'original reading' structs from the 'stored reading' structs, which are required for uploading to supabase
	originalReadings := d.repository.ConvertStoredToReadings(storedReadings)

	// TODO: organise error better
	uploadErr := d.supaClient.UploadReadings(originalReadings)
	if uploadErr != nil {
		uploadErr := fmt.Errorf("upload failed: %w", uploadErr)
		errInc := d.repository.IncrementUploadAttemptCount(storedReadings)
		if errInc != nil {
			return 0, fmt.Errorf("%w: increment upload attempt count: %w", uploadErr, errInc)
		}
		return 0, uploadErr
	}

	// TODO: what about failure here...

	deleteErr := d.repository.DeleteReadings(storedReadings)
	if deleteErr != nil {
		return 0, fmt.Errorf("delete bess readings (%+v): %w", storedReadings, deleteErr)
	}
	return len, nil
}
