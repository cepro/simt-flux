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
)

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

	siteMeter, err := acuvim2.New(
		config.SiteMeter.ID,
		config.SiteMeter.Host,
		config.SiteMeter.Pt1,
		config.SiteMeter.Pt2,
		config.SiteMeter.Ct1,
		config.SiteMeter.Ct2,
	)
	if err != nil {
		slog.Error("Failed to create site meter", "error", err)
		return
	}
	go siteMeter.Run(ctx, time.Millisecond*time.Duration(config.SiteMeter.PollIntervalMs))

	bessMeter, err := acuvim2.New(
		config.BessMeter.ID,
		config.BessMeter.Host,
		config.BessMeter.Pt1,
		config.BessMeter.Pt2,
		config.BessMeter.Ct1,
		config.BessMeter.Ct2,
	)
	if err != nil {
		slog.Error("Failed to create bess meter", "error", err)
		return
	}
	go bessMeter.Run(ctx, time.Millisecond*time.Duration(config.BessMeter.PollIntervalMs))

	powerPack, err := powerpack.New(config.Bess.ID, config.Bess.Host)
	if err != nil {
		slog.Error("Failed to create power pack", "error", err)
		return
	}
	go powerPack.Run(ctx, time.Millisecond*time.Duration(config.Bess.PollIntervalMs))

	dataPlatform, err := dataplatform.New(config.Supabase.Url, config.Supabase.Key, "telemetry.sqlite")
	if err != nil {
		slog.Error("Failed to create data platform", "error", err)
		return
	}
	go dataPlatform.Run(ctx)

	ctrl := controller.New(controller.Config{
		BessNameplatePower:     config.Bess.NameplatePower,
		BessNameplateEnergy:    config.Bess.NameplateEnergy,
		ImportAvoidancePeriods: config.Controller.ImportAvoidancePeriods,
		BessCommands:           powerPack.Commands,
	})
	go ctrl.Run(ctx, time.NewTicker(time.Second*5).C)

	// the meter and bess readings are sent to both the controller and the data platform
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case siteMeterReading := <-siteMeter.Telemetry:
				sendIfNonBlocking(ctrl.SiteMeterReadings, siteMeterReading, "Controller site meter readings")
				sendIfNonBlocking(dataPlatform.MeterReadings, siteMeterReading, "Dataplatform meter readings")
			// case bessMeterReading := <-bessMeter.Telemetry:
			// 	sendIfNonBlocking(dataPlatform.MeterReadings, bessMeterReading, "Dataplatform meter readings")
			case bessReading := <-powerPack.Telemetry:
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
