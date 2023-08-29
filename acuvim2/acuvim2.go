package acuvim2

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/cepro/besscontroller/telemetry"
	"github.com/google/uuid"
	"github.com/grid-x/modbus"
)

const (
	MODBUS_HOLDING_REGISTER_FREQ        = 12288
	MODBUS_HOLDING_REGISTER_TOTAL_POWER = 12322
)

// Acuvim2Meter handles Modbus communications with the three phase Acuvim 2 meters.
//
// Meter readings are taken regularly and sent onto the `Telemetry` channel.
type Acuvim2Meter struct {
	Telemetry chan telemetry.MeterReading

	host   string
	id     uuid.UUID
	pt1    float64
	pt2    float64
	ct1    float64
	ct2    float64
	client modbus.Client
}

func New(id uuid.UUID, host string, pt1 float64, pt2 float64, ct1 float64, ct2 float64) (*Acuvim2Meter, error) {

	meter := &Acuvim2Meter{
		Telemetry: make(chan telemetry.MeterReading),
		id:        id,
		host:      host,
		pt1:       pt1,
		pt2:       pt2,
		ct1:       ct1,
		ct2:       ct2,
	}

	handler := modbus.NewTCPClientHandler(host)
	handler.Timeout = 10 * time.Second
	handler.SlaveID = 0x01
	// handler.Logger = log.New(os.Stdout, "test: ", log.LstdFlags)

	fmt.Printf("Connecting to Acuvim meter %v...\n", host)

	err := handler.Connect()
	if err != nil {
		return nil, err
	}
	defer handler.Close()

	meter.client = modbus.NewClient(handler)

	fmt.Println("Connected")

	return meter, nil
}

// Run loops forever, polling telemetry from the meter every `period`.
func (a *Acuvim2Meter) Run(period time.Duration) error {

	readingTicker := time.NewTicker(period)

	for t := range readingTicker.C {

		frequency, err := a.pollFloat(MODBUS_HOLDING_REGISTER_FREQ)
		if err != nil {
			// TODO: more context to error messages an retries?
			return err
		}

		totalPower, err := a.pollFloat(MODBUS_HOLDING_REGISTER_TOTAL_POWER)
		if err != nil {
			return err
		}
		pt1 := 400.0
		pt2 := 400.0
		ct1 := 800.0
		ct2 := 5.0
		totalPower = (totalPower * (pt1 / pt2) * (ct1 / ct2)) / 1000.0

		a.Telemetry <- telemetry.MeterReading{
			ID:         uuid.New(),
			Time:       t,
			MeterID:    a.id,
			Frequency:  frequency,
			TotalPower: totalPower,
		}
	}

	return errors.New("Channel closed")
}

func (a *Acuvim2Meter) pollFloat(register uint16) (float64, error) {
	bytes, err := a.client.ReadHoldingRegisters(register, 2)
	if err != nil {
		return math.NaN(), err
	}
	return float64(float32FromBytes(bytes)), nil
}

func float32FromBytes(bytes []byte) float32 {
	valUint32 := binary.BigEndian.Uint32(bytes)
	valFloat32 := math.Float32frombits(valUint32)
	return valFloat32
}
