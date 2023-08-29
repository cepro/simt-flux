package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/cepro/besscontroller/acuvim2"
	"github.com/cepro/besscontroller/controller"
	dataplatform "github.com/cepro/besscontroller/data_platform"
	"github.com/cepro/besscontroller/tesla"
	timeutils "github.com/cepro/besscontroller/time_utils"
	"github.com/google/uuid"
)

func main() {

	fmt.Printf("Starting controller...\n")

	// TODO: read these kind of things from config file
	telemetryPollInterval := 1 * time.Second
	supabaseUrl := "https://hiffuporsxuzdmvgbtyp.supabase.co"
	supabaseKey := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImhpZmZ1cG9yc3h1emRtdmdidHlwIiwicm9sZSI6ImFub24iLCJpYXQiOjE2OTI3MTc5OTAsImV4cCI6MjAwODI5Mzk5MH0.4LBRWFK_qX0uu31uECrVqfMP8uGOCuTXr3DB3aA7zic"
	siteMeterID := uuid.MustParse("82b441ad-4475-4caf-a715-48bb86cebd96")
	bessMeterID := uuid.MustParse("7994fdcc-7dfa-4ef9-a529-e9167317ddb3")
	powePackID := uuid.MustParse("e2122808-1e75-4dd8-a67d-5a66ad54d433")
	location, err := time.LoadLocation("Europe/London")
	if err != nil {
		fmt.Printf("Failed to load time location: %v\n", err)
		return
	}

	// importAvoidancePeriods defines the time ranges that we should not import power
	importAvoidancePeriods := []timeutils.ClockTimePeriod{
		{
			Start: timeutils.ClockTime{Hour: 6, Minute: 0, Second: 0, Location: location},
			End:   timeutils.ClockTime{Hour: 15, Minute: 0, Second: 0, Location: location},
		},
		{
			Start: timeutils.ClockTime{Hour: 16, Minute: 0, Second: 0, Location: location},
			End:   timeutils.ClockTime{Hour: 19, Minute: 0, Second: 0, Location: location},
		},
	}

	// siteMeter, err := acuvim2.New(siteMeterID, "localhost:1502", 400, 400, 800, 5)
	siteMeter, err := acuvim2.NewEmulated(siteMeterID)
	if err != nil {
		fmt.Printf("Failed to create site meter: %v", err)
		return
	}
	go siteMeter.Run(telemetryPollInterval)

	// bessMeter, err := acuvim2.New(bessMeterID, "localhost:1503", 400, 400, 400, 5)
	bessMeter, err := acuvim2.NewEmulated(bessMeterID)
	if err != nil {
		fmt.Printf("Failed to create bess meter: %v", err)
		return
	}
	go bessMeter.Run(telemetryPollInterval)

	powerPack, err := tesla.NewPowerPack(powePackID, "localhost:1504")
	if err != nil {
		fmt.Printf("Failed to create power pack: %v", err)
		return
	}
	go powerPack.Run(telemetryPollInterval)

	dataPlatform, err := dataplatform.New(supabaseUrl, supabaseKey)
	if err != nil {
		fmt.Printf("Failed to create data platform: %v", err)
		return
	}
	go dataPlatform.Run()

	ctrl := controller.New(controller.Config{
		BessNameplatePower:     340e3,
		BessNameplateEnergy:    444e3,
		ImportAvoidancePeriods: importAvoidancePeriods,
		BessCommands:           powerPack.Commands,
	})
	go ctrl.Run()

	// the meter and bess readings are sent to both the controller and the data platform
	go func() {
		for {
			select {
			case siteMeterReading := <-siteMeter.Telemetry:
				dataPlatform.MeterReadings <- siteMeterReading
				ctrl.SiteMeterReadings <- siteMeterReading
			case bessMeterReading := <-bessMeter.Telemetry:
				dataPlatform.MeterReadings <- bessMeterReading
			case bessReading := <-powerPack.Telemetry:
				dataPlatform.BessReadings <- bessReading
				ctrl.BessReadings <- bessReading
			}

		}
	}()

	// wait for a ctrl-c interrupt before exiting
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
}
