package repository

import "github.com/cepro/besscontroller/telemetry"

type StoredMeterReading struct {
	telemetry.MeterReading
	UploadAttemptCount uint
}

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
