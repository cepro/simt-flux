package config

import (
	"fmt"
	"os"

	"github.com/cepro/besscontroller/cartesian"
	timeutils "github.com/cepro/besscontroller/time_utils"
	"github.com/google/uuid"
	"gopkg.in/yaml.v2"
)

type DynamicPeakDischargeConfig struct {
	DayedPeriod            timeutils.DayedPeriod        `yaml:"period"`
	TargetSoe              float64                      `yaml:"targetSoe"`
	TargetShortPeriods     bool                         `yaml:"targetShortPeriods"`
	ShortPrediction        NivPredictionDirectionConfig `yaml:"shortPrediction"`
	PrioritiseResidualLoad bool                         `yaml:"prioritiseResidualLoad"`
}

type DynamicPeakApproachConfig struct {
	PeakPeriod                    timeutils.DayedPeriod        `yaml:"peakPeriod"`
	ToSoe                         float64                      `yaml:"toSoe"`
	EncourageToSoe                float64                      `yaml:"encourageToSoe"`
	AssumedChargePower            float64                      `yaml:"assumedChargePower"`
	ForceChargeDurationFactor     float64                      `yaml:"forceChargeDurationFactor"`
	EncourageChargeDurationFactor float64                      `yaml:"encourageChargeDurationFactor"`
	ChargeCushionMins             float64                      `yaml:"chargeCushionMins"`
	LongPrediction                NivPredictionDirectionConfig `yaml:"longPrediction"`
}

func (c DynamicPeakDischargeConfig) GetDayedPeriod() timeutils.DayedPeriod {
	return c.DayedPeriod
}

type ImportAvoidanceWhenShortConfig struct {
	DayedPeriod     timeutils.DayedPeriod        `yaml:"period"`
	ShortPrediction NivPredictionDirectionConfig `yaml:"shortPrediction"`
}

func (c ImportAvoidanceWhenShortConfig) GetDayedPeriod() timeutils.DayedPeriod {
	return c.DayedPeriod
}

type DayedPeriodWithSoe struct {
	DayedPeriod timeutils.DayedPeriod `yaml:"period"`
	Soe         float64               `yaml:"soe"`
}

func (c DayedPeriodWithSoe) GetDayedPeriod() timeutils.DayedPeriod {
	return c.DayedPeriod
}

type NivConfig struct {
	ChargeCurve     cartesian.Curve     `yaml:"chargeCurve"`
	DischargeCurve  cartesian.Curve     `yaml:"dischargeCurve"`
	CurveShiftLong  float64             `yaml:"curveShiftLong"`
	CurveShiftShort float64             `yaml:"curveShiftShort"`
	DefaultPricing  []TimedRate         `yaml:"defaultPricing"`
	Prediction      NivPredictionConfig `yaml:"pricePrediction"`
}

type NivPredictionConfig struct {
	WhenShort NivPredictionDirectionConfig `yaml:"whenShort"`
	WhenLong  NivPredictionDirectionConfig `yaml:"whenLong"`
}

// TODO: think about naming here 'prediction' is used for both up to date modo and previous modo
// in the code, but in the config it's used only for previosu modo data
type NivPredictionDirectionConfig struct {
	AllowPrediction bool    `yaml:"allowPrediction"`
	VolumeCutoff    float64 `yaml:"volumeCutoff"` // imbalance volume in kWh
	TimeCutoffSecs  int     `yaml:"timeCutoffSecs"`
}

type DayedPeriodWithNIV struct {
	DayedPeriod timeutils.DayedPeriod `yaml:"period"`
	Niv         NivConfig             `yaml:"niv"`
}

func (c DayedPeriodWithNIV) GetDayedPeriod() timeutils.DayedPeriod {
	return c.DayedPeriod
}

type DeviceConfig struct {
	Host             string    `yaml:"host"`
	ID               uuid.UUID `yaml:"id"`
	PollIntervalSecs int       `yaml:"pollIntervalSecs"`
}

type MetersConfig struct {
	Acuvim2 map[string]Acuvim2MeterConfig `yaml:"acuvim2"`
	Mock    map[string]Acuvim2MeterConfig `yaml:"mock"`
}

type Acuvim2MeterConfig struct {
	DeviceConfig `yaml:",inline"`
	Pt1          float64 `yaml:"pt1"`
	Pt2          float64 `yaml:"pt2"`
	Ct1          float64 `yaml:"ct1"`
	Ct2          float64 `yaml:"ct2"`
}

type MockMeterConfig struct {
	DeviceConfig `yaml:",inline"`
}

type PowerPackConfig struct {
	DeviceConfig    `yaml:",inline"`
	NameplatePower  float64               `yaml:"nameplatePower"`
	NameplateEnergy float64               `yaml:"nameplateEnergy"`
	TeslaOptions    PowerPackTeslaOptions `yaml:"teslaOptions"`
}

// PowerPackTeslaOptions contains settings which are applied via Modbus onto the tesla hardware.
// (maybe this struct could have a better name).
type PowerPackTeslaOptions struct {
	InverterRampRateUp   float64 `yaml:"inverterRampRateUp"`
	InverterRampRateDown float64 `yaml:"inverterRampRateDown"`
	AlwaysActive         bool    `yaml:"alwaysActive"`
}

type MockBessConfig struct {
	DeviceConfig    `yaml:",inline"`
	NameplatePower  float64 `yaml:"nameplatePower"`
	NameplateEnergy float64 `yaml:"nameplateEnergy"`
}

type BessConfig struct {
	PowerPack *PowerPackConfig `yaml:"powerPack"`
	Mock      *MockBessConfig  `yaml:"mock"`
}

type SupabaseConfig struct {
	Url string `yaml:"url"`
	// key is specified via env var
	Schema        string `yaml:"schema"`
	AnonKeyEnvVar string `yaml:"anonKeyEnvVar"`
	UserKeyEnvVar string `yaml:"userKeyEnvVar"`
}

type DataPlatformConfig struct {
	UploadIntervalSecs int            `yaml:"uploadIntervalSecs"`
	Supabase           SupabaseConfig `yaml:"supabase"`
}

type EmulationConfig struct {
	BessIsEmulated    bool      `yaml:"bessIsEmulated"`
	EmulatedSiteMeter uuid.UUID `yaml:"emulatedSiteMeter"`
}

type ControlComponentsConfig struct {
	ImportAvoidancePeriods   []timeutils.DayedPeriod          `yaml:"importAvoidance"`
	ExportAvoidancePeriods   []timeutils.DayedPeriod          `yaml:"exportAvoidance"`
	ImportAvoidanceWhenShort []ImportAvoidanceWhenShortConfig `yaml:"importAvoidanceWhenShort"`
	ChargeToSoePeriods       []DayedPeriodWithSoe             `yaml:"chargeToSoe"`
	DischargeToSoePeriods    []DayedPeriodWithSoe             `yaml:"dischargeToSoe"`
	DynamicPeakDischarges    []DynamicPeakDischargeConfig     `yaml:"dynamicPeakDischarge"`
	DynamicPeakAproaches     []DynamicPeakApproachConfig      `yaml:"dynamicPeakApproach"`
	NivChasePeriods          []DayedPeriodWithNIV             `yaml:"nivChase"`
}

type ControllerConfig struct {
	SiteMeterID             uuid.UUID               `yaml:"siteMeter"`
	BessMeterID             uuid.UUID               `yaml:"bessMeter"`
	Emulation               EmulationConfig         `yaml:"emulation"`
	BessChargeEfficiency    float64                 `yaml:"bessChargeEfficiency"`
	BessSoeMin              float64                 `yaml:"bessSoeMin"`
	BessSoeMax              float64                 `yaml:"bessSoeMax"`
	BessChargePowerLimit    float64                 `yaml:"bessChargePowerLimit"`
	BessDischargePowerLimit float64                 `yaml:"bessDischargePowerLimit"`
	SiteImportPowerLimit    float64                 `yaml:"siteImportPowerLimit"`
	SiteExportPowerLimit    float64                 `yaml:"siteExportPowerLimit"`
	ControlComponents       ControlComponentsConfig `yaml:"controlComponents"`
	RatesImport             []TimedRate             `yaml:"ratesImport"`
	RatesExport             []TimedRate             `yaml:"ratesExport"`
}

type AxleConfig struct {
	Host                        string `yaml:"host"`
	UsernameEnvVar              string `yaml:"usernameEnvVar"`
	PasswordEnvVar              string `yaml:"passwordEnvVar"`
	TelemetryUploadIntervalSecs int    `yaml:"telemetryUploadIntervalSecs"`
	SchedulePollIntervalSecs    int    `yaml:"schedulePollIntervalSecs"`
}

type Config struct {
	Meters        MetersConfig         `yaml:"meters"`
	Bess          BessConfig           `yaml:"bess"`
	DataPlatforms []DataPlatformConfig `yaml:"dataPlatforms"`
	Axle          *AxleConfig          `yaml:"axle,omitempty"`
	Controller    ControllerConfig     `yaml:"controller"`
}

func Read(path string) (Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config file: %w", err)
	}

	var config Config
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	return config, nil
}
