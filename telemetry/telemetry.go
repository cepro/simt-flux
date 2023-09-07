package telemetry

import (
	"time"

	"github.com/google/uuid"
)

// ReadingMeta holds meta data about a reading
type ReadingMeta struct {
	ID       uuid.UUID // The identifier for this reading
	DeviceID uuid.UUID // The identifier for the device this reading came from - e.g. the meter ID or BESS ID
	Time     time.Time // The time that the reading *started* to be taken (e.g. the time that the first modbus request was initiated)
}

// BessReading holds data pulled from a battery energy storage system
type BessReading struct {
	ReadingMeta
	Soe         float64
	TargetPower float64
	// TODO: other data...
}

// MeterReading holds data pulled from a meter
type MeterReading struct {
	ReadingMeta
	Frequency            float64
	VoltageLineAverage   float64
	CurrentPhA           float64
	CurrentPhB           float64
	CurrentPhC           float64
	CurrentPhAverage     float64
	PowerPhAActive       float64
	PowerPhBActive       float64
	PowerPhCActive       float64
	PowerTotalActive     float64
	PowerTotalReactive   float64
	PowerTotalApparent   float64
	PowerFactorTotal     float64
	EnergyImportedActive float64
	EnergyExportedActive float64
}

// BessCommand holds control data that is sent to a battery energy storage system
type BessCommand struct {
	TargetPower float64
	// TODO: other data...
	// TODO: this is not really telemetry but it's currently in a package called telemetry...
}
