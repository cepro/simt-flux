package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cepro/besscontroller/cartesian"
	timeutils "github.com/cepro/besscontroller/time_utils"
	"github.com/google/uuid"
)

type ImportAvoidanceWhenShortConfig struct {
	Period          timeutils.DayedPeriod        `json:"period"`
	ShortPrediction NivPredictionDirectionConfig `json:"shortPrediction"`
}

type DayedPeriodWithSoe struct {
	Period timeutils.DayedPeriod `json:"period"`
	Soe    float64               `json:"soe"`
}

type NivConfig struct {
	ChargeCurve     cartesian.Curve     `json:"chargeCurve"`
	DischargeCurve  cartesian.Curve     `json:"dischargeCurve"`
	CurveShiftLong  float64             `json:"curveShiftLong"`
	CurveShiftShort float64             `json:"curveShiftShort"`
	DefaultPricing  []TimedRate         `json:"defaultPricing"`
	Prediction      NivPredictionConfig `json:"pricePrediction"`
}

type NivPredictionConfig struct {
	WhenShort NivPredictionDirectionConfig `json:"whenShort"`
	WhenLong  NivPredictionDirectionConfig `json:"whenLong"`
}

// TODO: think about naming here 'prediction' is used for both up to date modo and previous modo
// in the code, but in the config it's used only for previosu modo data
type NivPredictionDirectionConfig struct {
	AllowPrediction bool    `json:"allowPrediction"`
	VolumeCutoff    float64 `json:"volumeCutoff"` // imbalance volume in kWh
	TimeCutoffSecs  int     `json:"timeCutoffSecs"`
}

type DayedPeriodWithNIV struct {
	Period timeutils.DayedPeriod `json:"period"`
	Niv    NivConfig             `json:"niv"`
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

type PowerPackConfig struct {
	DeviceConfig
	NameplatePower  float64               `json:"nameplatePower"`
	NameplateEnergy float64               `json:"nameplateEnergy"`
	TeslaOptions    PowerPackTeslaOptions `json:"teslaOptions"`
}

// PowerPackTeslaOptions contains settings which are applied via Modbus onto the tesla hardware.
// (maybe this struct could have a better name).
type PowerPackTeslaOptions struct {
	InverterRampRateUp   float64 `json:"inverterRampRateUp"`
	InverterRampRateDown float64 `json:"inverterRampRateDown"`
	AlwaysActive         bool    `json:"alwaysActive"`
}

type MockBessConfig struct {
	DeviceConfig
	NameplatePower  float64 `json:"nameplatePower"`
	NameplateEnergy float64 `json:"nameplateEnergy"`
}

type BessConfig struct {
	PowerPack *PowerPackConfig `json:"powerPack"`
	Mock      *MockBessConfig  `json:"mock"`
}

type SupabaseConfig struct {
	Url string `json:"url"`
	// key is specified via env var
	Schema        string `json:"schema"`
	AnonKeyEnvVar string `json:"anonKeyEnvVar"`
	UserKeyEnvVar string `json:"userKeyEnvVar"`
}

type DataPlatformConfig struct {
	UploadIntervalSecs int            `json:"uploadIntervalSecs"`
	Supabase           SupabaseConfig `json:"supabase"`
}

type EmulationConfig struct {
	BessIsEmulated    bool      `json:"bessIsEmulated"`
	EmulatedSiteMeter uuid.UUID `json:"emulatedSiteMeter"`
}

type ControlComponentsConfig struct {
	ImportAvoidancePeriods   []timeutils.DayedPeriod          `json:"importAvoidancePeriods"`
	ExportAvoidancePeriods   []timeutils.DayedPeriod          `json:"exportAvoidancePeriods"`
	ImportAvoidanceWhenShort []ImportAvoidanceWhenShortConfig `json:"importAvoidanceWhenShort"`
	ChargeToSoePeriods       []DayedPeriodWithSoe             `json:"chargeToSoePeriods"`
	DischargeToSoePeriods    []DayedPeriodWithSoe             `json:"dischargeToSoePeriods"`
	NivChasePeriods          []DayedPeriodWithNIV             `json:"nivChasePeriods"`
}

type ControllerConfig struct {
	SiteMeterID             uuid.UUID               `json:"siteMeter"`
	BessMeterID             uuid.UUID               `json:"bessMeter"`
	Emulation               EmulationConfig         `json:"emulation"`
	BessChargeEfficiency    float64                 `json:"bessChargeEfficiency"`
	BessSoeMin              float64                 `json:"bessSoeMin"`
	BessSoeMax              float64                 `json:"bessSoeMax"`
	BessChargePowerLimit    float64                 `json:"bessChargePowerLimit"`
	BessDischargePowerLimit float64                 `json:"bessDischargePowerLimit"`
	SiteImportPowerLimit    float64                 `json:"siteImportPowerLimit"`
	SiteExportPowerLimit    float64                 `json:"siteExportPowerLimit"`
	ControlComponents       ControlComponentsConfig `json:"controlComponents"`
	RatesImport             []TimedRate             `json:"ratesImport"`
	RatesExport             []TimedRate             `json:"ratesExport"`
}

type Config struct {
	Meters        MetersConfig         `json:"meters"`
	Bess          BessConfig           `json:"bess"`
	DataPlatforms []DataPlatformConfig `json:"dataPlatforms"`
	Controller    ControllerConfig     `json:"controller"`
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
