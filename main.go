package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/cepro/besscontroller/acuvim2"
	"github.com/cepro/besscontroller/axleclient"
	"github.com/cepro/besscontroller/axlemgr"
	"github.com/cepro/besscontroller/config"
	"github.com/cepro/besscontroller/controller"
	dataplatform "github.com/cepro/besscontroller/data_platform"
	"github.com/cepro/besscontroller/modo"
	"github.com/cepro/besscontroller/powerpack"
	"github.com/cepro/besscontroller/telemetry"
	"github.com/google/uuid"
)

const (
	CONTROL_LOOP_PERIOD = time.Second * 4
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
	var bessID uuid.UUID
	if config.Bess.PowerPack != nil {
		ppConfig := config.Bess.PowerPack
		bessID = ppConfig.ID
		slog.Debug("Creating real powerpack", "bess_id", ppConfig.ID)
		powerPack, err := powerpack.New(
			ppConfig.ID,
			ppConfig.Host,
			ppConfig.NameplateEnergy,
			ppConfig.NameplatePower,
			powerpack.TeslaOptions{
				RampRateUp:       ppConfig.TeslaOptions.InverterRampRateUp,
				RampRateDown:     ppConfig.TeslaOptions.InverterRampRateDown,
				AlwaysActiveMode: ppConfig.TeslaOptions.AlwaysActive,
			},
		)
		if err != nil {
			slog.Error("Failed to create power pack", "error", err)
			return
		}
		bess = powerPack
		go powerPack.Run(ctx, time.Second*time.Duration(config.Bess.PowerPack.PollIntervalSecs))
	} else if config.Bess.Mock != nil {
		mockConfig := config.Bess.Mock
		bessID = mockConfig.ID
		slog.Debug("Creating mock powerpack", "bess_id", mockConfig.ID)
		powerPackMock, err := powerpack.NewMock(mockConfig.ID, mockConfig.NameplateEnergy, mockConfig.NameplatePower)
		if err != nil {
			slog.Error("Failed to create mock power pack", "error", err)
			return
		}
		bess = powerPackMock
		go powerPackMock.Run(ctx, time.Second*time.Duration(config.Bess.Mock.PollIntervalSecs))
	}

	// The configuration can define multiple "dataplatforms" - we upload telemetry to each one
	dataPlatforms := make([]*dataplatform.DataPlatform, 0, len(config.DataPlatforms))
	for _, dataPlatformConfig := range config.DataPlatforms {

		// use the supabase url to create a unique sqlite buffer filename
		bufferFilename := strings.TrimPrefix(dataPlatformConfig.Supabase.Url, "https://")
		bufferFilename = strings.TrimPrefix(bufferFilename, "http://")
		bufferFilename = fmt.Sprintf("telemetry_%s.sqlite", bufferFilename)

		// Read supabase key secrets from env vars
		supabaseAnonKey, ok := os.LookupEnv(dataPlatformConfig.Supabase.AnonKeyEnvVar)
		if !ok {
			slog.Error("Environment variable not found", "env_var", dataPlatformConfig.Supabase.AnonKeyEnvVar)
			return
		}
		supabaseUserKey, ok := os.LookupEnv(dataPlatformConfig.Supabase.UserKeyEnvVar)
		if !ok {
			slog.Error("Environment variable not found", "env_var", dataPlatformConfig.Supabase.UserKeyEnvVar)
			return
		}

		dataPlatform, err := dataplatform.New(
			dataPlatformConfig.Supabase.Url,
			supabaseAnonKey,
			supabaseUserKey,
			dataPlatformConfig.Supabase.Schema,
			bufferFilename,
		)
		if err != nil {
			slog.Error("Failed to create data platform", "supabase_url", dataPlatformConfig.Supabase.Url, "error", err)
			return
		}
		go dataPlatform.Run(ctx, time.Second*time.Duration(dataPlatformConfig.UploadIntervalSecs))
		dataPlatforms = append(dataPlatforms, dataPlatform)
	}

	// Create modo client
	// TODO: run retrieval immediately, otherwise we get "cannot run NIV chasing messages" when it first runs up
	modoClient := modo.New(http.Client{Timeout: time.Second * 10})
	go modoClient.Run(ctx, time.Second*30)

	ctrl := controller.New(controller.Config{
		BessIsEmulated:           config.Controller.Emulation.BessIsEmulated,
		BessChargeEfficiency:     config.Controller.BessChargeEfficiency,
		BessSoeMin:               config.Controller.BessSoeMin,
		BessSoeMax:               config.Controller.BessSoeMax,
		BessChargePowerLimit:     config.Controller.BessChargePowerLimit,
		BessDischargePowerLimit:  config.Controller.BessDischargePowerLimit,
		SiteImportPowerLimit:     config.Controller.SiteImportPowerLimit,
		SiteExportPowerLimit:     config.Controller.SiteExportPowerLimit,
		ImportAvoidancePeriods:   config.Controller.ControlComponents.ImportAvoidancePeriods,
		ExportAvoidancePeriods:   config.Controller.ControlComponents.ExportAvoidancePeriods,
		ImportAvoidanceWhenShort: config.Controller.ControlComponents.ImportAvoidanceWhenShort,
		ChargeToSoePeriods:       config.Controller.ControlComponents.ChargeToSoePeriods,
		DischargeToSoePeriods:    config.Controller.ControlComponents.DischargeToSoePeriods,
		DynamicPeakDischarges:    config.Controller.ControlComponents.DynamicPeakDischarges,
		DynamicPeakApproaches:    config.Controller.ControlComponents.DynamicPeakAproaches,
		NivChasePeriods:          config.Controller.ControlComponents.NivChasePeriods,
		RatesImport:              config.Controller.RatesImport,
		RatesExport:              config.Controller.RatesExport,
		ModoClient:               modoClient,
		MaxReadingAge:            CONTROL_LOOP_PERIOD,
		BessCommands:             bess.Commands(),
	})
	go ctrl.Run(ctx, time.NewTicker(CONTROL_LOOP_PERIOD).C)

	// Create the Axle API if it's configured
	var axleManager *axlemgr.AxleMgr
	if config.Axle != nil {

		axleUsername, ok := os.LookupEnv(config.Axle.UsernameEnvVar)
		if !ok {
			slog.Error("Environment variable not found", "env_var", config.Axle.UsernameEnvVar)
			return
		}
		axlePassword, ok := os.LookupEnv(config.Axle.PasswordEnvVar)
		if !ok {
			slog.Error("Environment variable not found", "env_var", config.Axle.PasswordEnvVar)
			return
		}

		axleClient := axleclient.New(
			http.Client{Timeout: time.Second * 10},
			config.Axle.Host,
			axleUsername,
			axlePassword,
		)

		axleManager = axlemgr.New(
			ctrl.AxleSchedules,
			axleClient,
			bessID,
			config.Controller.SiteMeterID,
			config.Controller.BessMeterID,
			config.Axle.AssetId,
			bess.NameplateEnergy(),
		)

		go axleManager.Run(
			ctx,
			time.Second*time.Duration(config.Axle.TelemetryUploadIntervalSecs),
			time.Second*time.Duration(config.Axle.SchedulePollIntervalSecs),
		)
	}

	// fan out the meter and bess readings to various modules: the controller, the data platform, and Axle API
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case meterReading := <-meterReadings:
				if meterReading.DeviceID == config.Controller.SiteMeterID {

					sendIfNonBlocking(ctrl.SiteMeterReadings, meterReading, "Controller site meter readings")

					if config.Controller.Emulation.BessIsEmulated {
						sendIfNonBlocking(meterReadings, emulateSiteMeterReading(config.Controller.Emulation.EmulatedSiteMeter, ctrl, meterReading), "Emulated meter reading")
					}
				}
				for _, dataPlatform := range dataPlatforms {
					sendIfNonBlocking(dataPlatform.MeterReadings, meterReading, fmt.Sprintf("Dataplatform meter readings (%s)", dataPlatform.BufferRepositoryFilename()))
				}
				if axleManager != nil {
					sendIfNonBlocking(axleManager.MeterReadings, meterReading, "Axle meter readings")
				}
			case bessReading := <-bess.Telemetry():
				sendIfNonBlocking(ctrl.BessReadings, bessReading, "Controller bess readings")
				for _, dataPlatform := range dataPlatforms {
					sendIfNonBlocking(dataPlatform.BessReadings, bessReading, fmt.Sprintf("Dataplatform bess readings (%s)", dataPlatform.BufferRepositoryFilename()))
				}
				if axleManager != nil {
					sendIfNonBlocking(axleManager.BessReadings, bessReading, "Axle bess readings")
				}
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

// emulateSiteMeter generates a new emulated meter reading for every 'real' site meter reading. The emulated reading shows what the site power would be
// if the bess was really delivering power.
func emulateSiteMeterReading(emulatedSiteMeter uuid.UUID, ctrl *controller.Controller, meterReading telemetry.MeterReading) telemetry.MeterReading {
	emulatedPower := ctrl.EmulatedSitePower()
	return telemetry.MeterReading{
		ReadingMeta: telemetry.ReadingMeta{
			ID:       uuid.New(),
			DeviceID: emulatedSiteMeter,
			Time:     meterReading.Time,
		},
		PowerTotalActive: &emulatedPower,
	}
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
