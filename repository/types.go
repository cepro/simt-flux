package repository

import "github.com/cepro/besscontroller/telemetry"

// StoredMeterReading represents a meter reading that is persisted to the SQLite database, and includes a count of upload attempts.
type StoredMeterReading struct {
	telemetry.MeterReading
	UploadAttemptCount uint
}

// StoredBessReading represents a BESS reading that is persisted to the SQLite database, and includes a count of upload attempts.
type StoredBessReading struct {
	telemetry.BessReading
	UploadAttemptCount uint
}

func newStoredMeterReading(reading telemetry.MeterReading) StoredMeterReading {
	return StoredMeterReading{
		MeterReading:       reading,
		UploadAttemptCount: 0,
	}
}

func newStoredBessReading(reading telemetry.BessReading) StoredBessReading {
	return StoredBessReading{
		BessReading:        reading,
		UploadAttemptCount: 0,
	}
}
