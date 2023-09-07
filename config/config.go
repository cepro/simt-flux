package config

import (
	"encoding/json"
	"fmt"
	"os"

	timeutils "github.com/cepro/besscontroller/time_utils"
	"github.com/google/uuid"
)

type DeviceConfig struct {
	Host string    `json:"host"`
	ID   uuid.UUID `json:"id"`
}

type Acuvim2MeterConfig struct {
	DeviceConfig
	Pt1 float64 `json:"pt1"`
	Pt2 float64 `json:"pt2"`
	Ct1 float64 `json:"ct1"`
	Ct2 float64 `json:"ct2"`
}

type BessConfig struct {
	DeviceConfig
	NameplatePower  float64 `json:"nameplatePower"`
	NameplateEnergy float64 `json:"nameplateEnergy"`
}

type SupabaseConfig struct {
	Url string `json:"url"`
	Key string `json:"key"`
}

type ControllerConfig struct {
	Timezone               string                      `json:"timezone"`
	ImportAvoidancePeriods []timeutils.ClockTimePeriod `json:"importAvoidancePeriods"`
}

type Config struct {
	SiteMeter  Acuvim2MeterConfig `json:"siteMeter"`
	BessMeter  Acuvim2MeterConfig `json:"bessMeter"`
	Bess       BessConfig         `json:"bess"`
	Supabase   SupabaseConfig     `json:"supabase"`
	Controller ControllerConfig   `json:"controller"`
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
