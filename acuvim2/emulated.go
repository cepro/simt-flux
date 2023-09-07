package acuvim2

// import (
// 	"context"
// 	"time"

// 	"github.com/cepro/besscontroller/telemetry"
// 	"github.com/google/uuid"
// )

// // EmulatedAcuvim2Meter looks like an Acuvim2Meter but produces fake data.
// type EmulatedAcuvim2Meter struct {
// 	Telemetry chan telemetry.MeterReading
// 	id        uuid.UUID
// }

// func NewEmulated(id uuid.UUID) (*EmulatedAcuvim2Meter, error) {

// 	meter := &EmulatedAcuvim2Meter{
// 		Telemetry: make(chan telemetry.MeterReading),
// 		id:        id,
// 	}
// 	return meter, nil
// }

// func (a *EmulatedAcuvim2Meter) Run(ctx context.Context, period time.Duration) error {

// 	readingTicker := time.NewTicker(period)

// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return ctx.Err()
// 		case t := <-readingTicker.C:

// 			frequency := 50.0
// 			totalPower := 10e3

// 			a.Telemetry <- telemetry.MeterReading{
// 				ReadingMeta: telemetry.ReadingMeta{
// 					ReadingID: uuid.New(),
// 					DeviceID:  a.id,
// 					Time:      t,
// 				},
// 				Frequency:        frequency,
// 				TotalActivePower: totalPower,
// 			}
// 		}
// 	}
// }
