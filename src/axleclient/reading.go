package axleclient

import "time"

// Reading holds telemetry to be sent to Axle and defines how it maps to JSON
type Reading struct {
	AssetId        string    `json:"asset_id"`
	StartTimestamp time.Time `json:"start_timestamp"`
	EndTimestamp   time.Time `json:"end_timestamp"`
	Value          float64   `json:"value"`
	Label          string    `json:"label"`
}
