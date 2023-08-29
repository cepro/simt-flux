package telemetry

import (
	"time"

	"github.com/google/uuid"
)

// MeterReading holds data pulled from a meter
type MeterReading struct {
	ID         uuid.UUID
	Time       time.Time
	MeterID    uuid.UUID
	Frequency  float64
	TotalPower float64
	// TODO: include three phase elemenets of voltage, power and current etc
	// TODO: include start/end times of reading?
	// TODO: should we differentiate between the types of meter - this is a three phase 'industrial' meter, but there are lots of Emlites on site too.
}

// BessReading holds data pulled from a battery energy storage system
type BessReading struct {
	ID          uuid.UUID
	Time        time.Time
	BessID      uuid.UUID
	Soe         float64
	TargetPower float64
	// TODO: other data...
}

// BessCommand holds control data that is sent to a battery energy storage system
type BessCommand struct {
	TargetPower float64
	// TODO: other data...
	// TODO: this is not really telemetry but it's currently in a package called telemetry...
}
