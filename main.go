package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/cepro/besscontroller/acuvim2"
	"github.com/cepro/besscontroller/config"
	"github.com/cepro/besscontroller/controller"
	dataplatform "github.com/cepro/besscontroller/data_platform"
	"github.com/cepro/besscontroller/powerpack"
	"github.com/cepro/besscontroller/telemetry"
	"github.com/google/uuid"
)

const (
	CONTROL_LOOP_PERIOD = time.Second * 5
)

type Bess interface {
	Run(ctx context.Context, period time.Duration) error
	NameplateEnergy() float64
	NameplatePower() float64
	Commands() chan<- telemetry.BessCommand
	Telemetry() <-chan telemetry.BessReading
}

func main() {

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	var configFilePath string
	flag.StringVar(&configFilePath, "f", "./config.json", "Specify config file path")
	flag.Parse()

	slog.Info("Starting", "config_file", configFilePath)

	config, err := config.Read(configFilePath)
	if err != nil {
		slog.Error("Failed to read config", "error", err)
		return
	}

	// Read secrets from env vars
	supabaseAnonKey, ok := os.LookupEnv("SUPABASE_ANON_KEY")
	if !ok {
		slog.Error("SUPABASE_ANON_KEY environment variable not specified")
		return
	}
	supabaseUserKey := os.Getenv("SUPABASE_USER_KEY")
	if !ok {
		slog.Error("SUPABASE_USER_KEY environment variable not specified")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	meterReadings := make(chan telemetry.MeterReading, 5)

	// Create Acuvim2 'real' meters
	acuvimMeters := make(map[uuid.UUID]*acuvim2.Acuvim2Meter, len(config.Meters.Acuvim2))
	for _, meterConfig := range config.Meters.Acuvim2 {
		slog.Debug("Creating real acuvim2 meter", "meter_id", meterConfig.ID)
		meter, err := acuvim2.New(
			meterReadings,
			meterConfig.ID,
			meterConfig.Host,
			meterConfig.Pt1,
			meterConfig.Pt2,
			meterConfig.Ct1,
			meterConfig.Ct2,
		)
		if err != nil {
			slog.Error("Failed to create meter", "meter_id", meterConfig.ID, "error", err)
			return
		}
		go meter.Run(ctx, time.Second*time.Duration(meterConfig.PollIntervalSecs))
		acuvimMeters[meterConfig.ID] = meter
	}

	// Create Acuvim2 mock meters
	mockMeters := make(map[uuid.UUID]*acuvim2.Acuvim2MeterMock, len(config.Meters.Mock))
	for _, meterConfig := range config.Meters.Mock {
		slog.Debug("Creating mock meter", "meter_id", meterConfig.ID)
		meter, err := acuvim2.NewMock(
			meterReadings,
			meterConfig.ID,
		)
		if err != nil {
			slog.Error("Failed to create mock meter", "meter_id", meterConfig.ID, "error", err)
			return
		}
		go meter.Run(ctx, time.Second*time.Duration(meterConfig.PollIntervalSecs))
		mockMeters[meterConfig.ID] = meter
	}

	var bess Bess
	if config.Bess.PowerPack != nil {
		ppConfig := config.Bess.PowerPack
		slog.Debug("Creating real powerpack", "bess_id", ppConfig.ID)
		powerPack, err := powerpack.New(
			ppConfig.ID,
			ppConfig.Host,
			ppConfig.NameplateEnergy,
			ppConfig.NameplatePower,
			ppConfig.InverterRampRateUp,
			ppConfig.InverterRampRateDown,
		)
		if err != nil {
			slog.Error("Failed to create power pack", "error", err)
			return
		}
		bess = powerPack
		go powerPack.Run(ctx, time.Second*time.Duration(config.Bess.PowerPack.PollIntervalSecs))
	} else if config.Bess.Mock != nil {
		mockConfig := config.Bess.Mock
		slog.Debug("Creating mock powerpack", "bess_id", mockConfig.ID)
		powerPackMock, err := powerpack.NewMock(mockConfig.ID, mockConfig.NameplateEnergy, mockConfig.NameplatePower)
		if err != nil {
			slog.Error("Failed to create mock power pack", "error", err)
			return
		}
		bess = powerPackMock
		go powerPackMock.Run(ctx, time.Second*time.Duration(config.Bess.Mock.PollIntervalSecs))
	}

	dataPlatform, err := dataplatform.New(config.DataPlatform.Supabase.Url, supabaseAnonKey, supabaseUserKey, config.DataPlatform.Supabase.Schema, "telemetry.sqlite")
	if err != nil {
		slog.Error("Failed to create data platform", "error", err)
		return
	}
	go dataPlatform.Run(ctx, time.Second*time.Duration(config.DataPlatform.UploadIntervalSecs))

	ctrl := controller.New(controller.Config{
		BessNameplatePower:     bess.NameplatePower(),
		BessNameplateEnergy:    bess.NameplateEnergy(),
		BessIsEmulated:         config.Controller.Emulation.BessIsEmulated,
		BessSoeMin:             config.Controller.BessSoeMin,
		BessSoeMax:             config.Controller.BessSoeMax,
		BessChargeEfficiency:   config.Controller.BessChargeEfficiency,
		ImportAvoidancePeriods: config.Controller.ImportAvoidancePeriods,
		ExportAvoidancePeriods: config.Controller.ExportAvoidancePeriods,
		ChargeToMinPeriods:     config.Controller.ChargeToMinPeriods,
		MaxReadingAge:          CONTROL_LOOP_PERIOD,
		BessCommands:           bess.Commands(),
	})
	go ctrl.Run(ctx, time.NewTicker(CONTROL_LOOP_PERIOD).C)

	// the meter and bess readings are sent to both the controller and the data platform
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case meterReading := <-meterReadings:
				if meterReading.DeviceID == config.Controller.SiteMeterID {

					sendIfNonBlocking(ctrl.SiteMeterReadings, meterReading, "Controller site meter readings")

					// If the bess is emulated then for every 'real' site meter reading we generate a new emulated meter reading, which shows what the site power would be
					// if the bess was really delivering power
					if config.Controller.Emulation.BessIsEmulated {
						emulatedPower := ctrl.EmulatedSitePower()
						emulatedReading := telemetry.MeterReading{
							ReadingMeta: telemetry.ReadingMeta{
								ID:       uuid.New(),
								DeviceID: config.Controller.Emulation.EmulatedSiteMeter,
								Time:     meterReading.Time,
							},
							PowerTotalActive: &emulatedPower,
						}
						sendIfNonBlocking(meterReadings, emulatedReading, "Emulated meter reading")
					}
				}
				sendIfNonBlocking(dataPlatform.MeterReadings, meterReading, "Dataplatform meter readings")
			case bessReading := <-bess.Telemetry():
				sendIfNonBlocking(ctrl.BessReadings, bessReading, "Controller bess readings")
				sendIfNonBlocking(dataPlatform.BessReadings, bessReading, "Dataplatform bess readings")
			}
		}
	}()

	// wait for a ctrl-c interrupt before exiting
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan

	// cancel any open go-routines and give them up to 100ms to gracefully shutdown
	cancel()
	time.Sleep(time.Millisecond * 100)

	slog.Info("Exiting")
	os.Exit(0)
}

// sendIfNonBlocking attempts to send the given value onto the given channel, but will only do so if the operation
// is non-blocking, otherwise it logs a warning message and returns.
func sendIfNonBlocking[V any](ch chan V, val V, messageTargetLogStr string) {
	select {
	case ch <- val:
	default:
		slog.Warn("Dropped message", "message_target", messageTargetLogStr)
	}
}
