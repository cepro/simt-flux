package repository

import (
	"fmt"

	"github.com/cepro/besscontroller/telemetry"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// repository stores telemetry to the local file system (sqlite) before it is uploaded to Supbase.
type Repository struct {
	db *gorm.DB
}

func New(path string) (*Repository, error) {

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	// Migrate the schema
	err = db.AutoMigrate(&StoredBessReading{}, &StoredMeterReading{})
	if err != nil {
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	return &Repository{
		db: db,
	}, nil
}

// convertReadingsForStorage returns the equivilent "stored type" (which includes an 'upload attempt count') for the given readings
func (r *Repository) convertReadingsForStorage(readings interface{}) interface{} {
	switch readingsTyped := readings.(type) {

	case []telemetry.BessReading:
		storedReading := make([]StoredBessReading, 0, len(readingsTyped))
		for _, reading := range readingsTyped {
			storedReading = append(storedReading, newStoredBessReading(reading))
		}
		return storedReading

	case []telemetry.MeterReading:
		storedReading := make([]StoredMeterReading, 0, len(readingsTyped))
		for _, reading := range readingsTyped {
			storedReading = append(storedReading, newStoredMeterReading(reading))
		}
		return storedReading

	default:
		panic(fmt.Sprintf("Unknown readings type: '%T'", readings))
	}
}

// ConvertStoredToReadings returns the "original reading" from teh given stored readings
func (r *Repository) ConvertStoredToReadings(storedReadings interface{}) interface{} {
	switch storedReadingsTyped := storedReadings.(type) {

	case []StoredBessReading:
		readings := make([]telemetry.BessReading, 0, len(storedReadingsTyped))
		for _, storedReading := range storedReadingsTyped {
			readings = append(readings, storedReading.BessReading)
		}
		return readings

	case []StoredMeterReading:
		readings := make([]telemetry.MeterReading, 0, len(storedReadingsTyped))
		for _, storedReading := range storedReadingsTyped {
			readings = append(readings, storedReading.MeterReading)
		}
		return readings

	default:
		panic(fmt.Sprintf("Unknown stored readings type: '%T'", storedReadings))
	}
}

// StoreReadings adds the given readings (which can be of any reading type) into the database and
// sets the 'upload attempt count' to 1.
func (r *Repository) StoreReadings(readings interface{}) error {
	storedReadings := r.convertReadingsForStorage(readings)
	result := r.db.Create(storedReadings)
	return result.Error
}

func (r *Repository) DeleteReadings(readings interface{}) error {
	result := r.db.Delete(&readings)
	return result.Error
}

func (r *Repository) GetMeterReadings(limit int) ([]StoredMeterReading, error) {
	var readings []StoredMeterReading

	query := r.db.Limit(limit).Order("upload_attempt_count asc, time desc")
	result := query.Find(&readings)
	if result.Error != nil {
		return nil, result.Error
	}
	return readings, nil
}

func (r *Repository) GetBessReadings(limit int) ([]StoredBessReading, error) {
	var readings []StoredBessReading

	// TODO: do we want to give up after a certain amount of attempts?
	query := r.db.Limit(limit).Order("upload_attempt_count asc, time desc")
	result := query.Find(&readings)
	if result.Error != nil {
		return nil, result.Error
	}
	return readings, nil
}

func (r *Repository) IncrementUploadAttemptCount(readings interface{}) error {
	result := r.db.Model(readings).UpdateColumn("upload_attempt_count", gorm.Expr("upload_attempt_count + ?", 1))
	return result.Error
}
