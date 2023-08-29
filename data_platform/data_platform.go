package dataplatform

import (
	"fmt"
	"time"

	"github.com/cepro/besscontroller/repository"
	"github.com/cepro/besscontroller/telemetry"

	supa "github.com/nedpals/supabase-go"
)

// DataPlatform handles the streaming of telemetry to Supabase.
// Put new meter and bess readings onto the appropriate channels and they will be uploaded.
type DataPlatform struct {
	MeterReadings chan telemetry.MeterReading
	BessReadings  chan telemetry.BessReading

	repository *repository.Repository
	supaClient *supa.Client
}

func New(supabaseUrl string, supabaseKey string) (*DataPlatform, error) {

	supaClient := supa.CreateClient(supabaseUrl, supabaseKey)

	repository, err := repository.New("telemetry.db")
	if err != nil {
		return nil, fmt.Errorf("create repository: %w", err)
	}

	return &DataPlatform{
		// TODO: properly consider bufferring behaviour of channels
		MeterReadings: make(chan telemetry.MeterReading, 25),
		BessReadings:  make(chan telemetry.BessReading, 25),
		repository:    repository,
		supaClient:    supaClient,
	}, nil
}

// Run loops forever waiting for meter or bess readings, when they are available they are uploaded.
func (d *DataPlatform) Run() {

	// TODO: consider alerting for best-efford approach

	uploadTicker := time.NewTicker(time.Second * 5)

	for {
		select {
		case reading := <-d.MeterReadings:
			err := d.repository.AddMeterReading(reading)
			if err != nil {
				fmt.Printf("failed to persist meter reading: %v\n", err)
			}
			fmt.Printf("Stored meter reading\n")

		case reading := <-d.BessReadings:
			err := d.repository.AddBessReading(reading)
			if err != nil {
				fmt.Printf("failed to persist bess reading: %v\n", err)
			}
			fmt.Printf("Stored bess reading\n")
		case _ = <-uploadTicker.C:
			d.attemptUploadPending()
		}
	}
}

func (d *DataPlatform) attemptUploadPending() {

	// uploadChunkLimit defines how many data points we can upload in one supabase HTTP request
	uploadChunkLimit := 50

	// first attempt to upload any new readings that have not been seen before
	freshMeterReadings, err := d.repository.GetMeterReadings(uploadChunkLimit, true)
	if err != nil {
		fmt.Printf("failed to query fresh meter readings: %v\n", err)
	} else {
		err = d.handleMeterReadings(freshMeterReadings)
		if err != nil {
			fmt.Printf("failed to handle fresh meter readings: %v\n", err)
		}
	}
	freshBessReadings, err := d.repository.GetBessReadings(uploadChunkLimit, true)
	if err != nil {
		fmt.Printf("failed to query fresh bess readings: %v\n", err)
	} else {
		err = d.handleBessReadings(freshBessReadings)
		if err != nil {
			fmt.Printf("failed to handle fresh bess readings: %v\n", err)
		}
	}

	// then attempt to upload any old meter readings that have already failed an upload at least once
	oldMeterReadings, err := d.repository.GetMeterReadings(uploadChunkLimit, false)
	if err != nil {
		fmt.Printf("failed to query old meter readings: %v\n", err)
	} else {
		err = d.handleMeterReadings(oldMeterReadings)
		if err != nil {
			fmt.Printf("failed to handle old meter readings: %v\n", err)
		}
	}
	oldBessReadings, err := d.repository.GetBessReadings(uploadChunkLimit, false)
	if err != nil {
		fmt.Printf("failed to query old meter readings: %v\n", err)
	} else {
		err = d.handleBessReadings(oldBessReadings)
		if err != nil {
			fmt.Printf("failed to handle old meter readings: %v\n", err)
		}
	}

	return
}

func (d *DataPlatform) handleMeterReadings(readings []repository.StoredMeterReading) error {

	if len(readings) < 1 {
		return nil
	}

	// rows = append(rows, supabaseMeterReading{})

	uploadErr := d.supaClient.DB.From("meter_readings").Insert(convertMeterReadings(readings)).Execute(nil)
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
		return fmt.Errorf("delete meter readings: %w", deleteErr)
	}

	// TODO: really think through this logic to handle edge cases, e.g. where the upload succeeds but the delete doesn't

	return nil
}

func (d *DataPlatform) handleBessReadings(readings []repository.StoredBessReading) error {

	if len(readings) < 1 {
		return nil
	}

	// rows = append(rows, supabaseBessReading{})

	uploadErr := d.supaClient.DB.From("bess_readings").Insert(convertBessReadings(readings)).Execute(nil)
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
		return fmt.Errorf("delete meter readings: %w", deleteErr)
	}

	// TODO: really think through this logic to handle edge cases, e.g. where the upload succeeds but the delete doesn't

	return nil
}

func (d *DataPlatform) uploadBessReadings(readings []telemetry.BessReading) error {
	var rows []supabaseBessReading
	for _, reading := range readings {
		rows = append(rows, supabaseBessReading(reading))
	}

	err := d.supaClient.DB.From("bess_readings").Insert(rows).Execute(nil)
	if err != nil {
		return err
	}

	return nil
}
