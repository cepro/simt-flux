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

func (r *Repository) AddMeterReading(reading telemetry.MeterReading) error {
	result := r.db.Create(newStoredMeterReading(reading))
	return result.Error
}

func (r *Repository) AddBessReading(reading telemetry.BessReading) error {
	result := r.db.Create(newStoredBessReading(reading))
	return result.Error
}

func (r *Repository) DeleteReadings(readings interface{}) error {
	result := r.db.Delete(&readings)
	return result.Error
}

func (r *Repository) GetMeterReadings(limit int, fresh bool) ([]StoredMeterReading, error) {
	var readings []StoredMeterReading

	query := r.db.Limit(limit).Order("upload_attempt_count asc, time desc")
	if fresh {
		query = query.Where("upload_attempt_count = ?", 0)
	} else {
		query = query.Where("upload_attempt_count > ?", 0)
		// TODO: do we want to give up after a certain amount of attempts?
	}
	result := query.Find(&readings)
	if result.Error != nil {
		return nil, result.Error
	}
	return readings, nil
}

func (r *Repository) GetBessReadings(limit int, fresh bool) ([]StoredBessReading, error) {
	var readings []StoredBessReading

	query := r.db.Limit(limit).Order("upload_attempt_count asc, time desc")
	if fresh {
		query = query.Where("upload_attempt_count = ?", 0)
	} else {
		query = query.Where("upload_attempt_count > ?", 0)
		// TODO: do we want to give up after a certain amount of attempts?
	}
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
