package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cepro/besscontroller/cartesian"
	timeutils "github.com/cepro/besscontroller/time_utils"
	"github.com/google/uuid"
)

type ClockTimePeriodWithSoe struct {
	Period timeutils.ClockTimePeriod `json:"period"`
	Soe    float64                   `json:"soe"`
}

type DeviceConfig struct {
	Host             string    `json:"host"`
	ID               uuid.UUID `json:"id"`
	PollIntervalSecs int       `json:"pollIntervalSecs"`
}

type MetersConfig struct {
	Acuvim2 map[string]Acuvim2MeterConfig `json:"acuvim2"`
	Mock    map[string]Acuvim2MeterConfig `json:"mock"`
}

type Acuvim2MeterConfig struct {
	DeviceConfig
	Pt1 float64 `json:"pt1"`
	Pt2 float64 `json:"pt2"`
	Ct1 float64 `json:"ct1"`
	Ct2 float64 `json:"ct2"`
}

type MockMeterConfig struct {
	DeviceConfig
}

type PowerPackBessConfig struct {
	DeviceConfig
	NameplatePower       float64 `json:"nameplatePower"`
	NameplateEnergy      float64 `json:"nameplateEnergy"`
	InverterRampRateUp   float64 `json:"inverterRampRateUp"`
	InverterRampRateDown float64 `json:"inverterRampRateDown"`
}

type MockBessConfig struct {
	DeviceConfig
	NameplatePower  float64 `json:"nameplatePower"`
	NameplateEnergy float64 `json:"nameplateEnergy"`
}

type BessConfig struct {
	PowerPack *PowerPackBessConfig `json:"powerPack"`
	Mock      *MockBessConfig      `json:"mock"`
}

type SupabaseConfig struct {
	Url string `json:"url"`
	// key is specified via env var
	Schema string `json:"schema"`
}

type DataPlatformConfig struct {
	UploadIntervalSecs int            `json:"uploadIntervalSecs"`
	Supabase           SupabaseConfig `json:"supabase"`
}

type EmulationConfig struct {
	BessIsEmulated    bool      `json:"bessIsEmulated"`
	EmulatedSiteMeter uuid.UUID `json:"emulatedSiteMeter"`
}

type ControllerConfig struct {
	SiteMeterID             uuid.UUID                   `json:"siteMeter"`
	BessMeterID             uuid.UUID                   `json:"bessMeter"`
	Emulation               EmulationConfig             `json:"emulation"`
	BessChargeEfficiency    float64                     `json:"bessChargeEfficiency"`
	BessSoeMin              float64                     `json:"bessSoeMin"`
	BessSoeMax              float64                     `json:"bessSoeMax"`
	BessChargePowerLimit    float64                     `json:"bessChargePowerLimit"`
	BessDischargePowerLimit float64                     `json:"bessDischargePowerLimit"`
	SiteImportPowerLimit    float64                     `json:"siteImportPowerLimit"`
	SiteExportPowerLimit    float64                     `json:"siteExportPowerLimit"`
	ImportAvoidancePeriods  []timeutils.ClockTimePeriod `json:"importAvoidancePeriods"`
	ExportAvoidancePeriods  []timeutils.ClockTimePeriod `json:"exportAvoidancePeriods"`
	ChargeToSoePeriods      []ClockTimePeriodWithSoe    `json:"chargeToSoePeriods"`
	DischargeToSoePeriods   []ClockTimePeriodWithSoe    `json:"dischargeToSoePeriods"`
	NivChasePeriods         []timeutils.ClockTimePeriod `json:"nivChasePeriods"`
	NivChargeCurve          []cartesian.Point           `json:"nivChargeCurve"`
	NivDischargeCurve       []cartesian.Point           `json:"nivDischargeCurve"`
	NivCurveShiftLong       float64                     `json:"nivCurveShiftLong"`
	NivCurveShiftShort      float64                     `json:"nivCurveShiftShort"`
	NivDefaultPricing       []TimedCharge               `json:"nivDefaultPricing"`
	DuosChargesImport       []TimedCharge               `json:"duosChargesImport"`
	DuosChargesExport       []TimedCharge               `json:"duosChargesExport"`
}

type TimedCharge struct {
	Rate           float64                     `json:"rate"`
	PeriodsWeekday []timeutils.ClockTimePeriod `json:"weekdayPeriods"`
	PeriodsWeekend []timeutils.ClockTimePeriod `json:"weekendPeriods"`
}

type Config struct {
	Meters       MetersConfig       `json:"meters"`
	Bess         BessConfig         `json:"bess"`
	DataPlatform DataPlatformConfig `json:"dataPlatform"`
	Controller   ControllerConfig   `json:"controller"`
}

func Read(path string) (Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config file: %w", err)
	}

	var config Config
	err = json.Unmarshal(content, &config)
	if err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	return config, nil
}
