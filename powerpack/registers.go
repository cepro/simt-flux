package powerpack

import "github.com/cepro/besscontroller/modbus"

var configBlock = modbus.MetricBlock{
	Name:         "Config",
	StartAddr:    100,
	NumRegisters: 47,
	Metrics: map[string]modbus.Metric{

		"ProtocolVersion": {
			StartAddr:   100,
			DataType:    modbus.Int16Type,
			ScalingFunc: nil,
		},
		"FirmwareVersion": {
			StartAddr:   102,
			DataType:    modbus.String32Type,
			ScalingFunc: nil,
		},
		"Serial": {
			StartAddr:   118,
			DataType:    modbus.String32Type,
			ScalingFunc: nil,
		},
		"NumBattMeters": {
			StartAddr:   134,
			DataType:    modbus.Int16Type,
			ScalingFunc: nil,
		},
		"NumSiteMeters": {
			StartAddr:   135,
			DataType:    modbus.Int16Type,
			ScalingFunc: nil,
		},
		"MaxChargePower": {
			StartAddr:   139,
			DataType:    modbus.Int32Type,
			ScalingFunc: nil,
		},
		"MaxDischargePower": {
			StartAddr:   141,
			DataType:    modbus.Int32Type,
			ScalingFunc: nil,
		},
		"Energy": {
			StartAddr:   145,
			DataType:    modbus.Int32Type,
			ScalingFunc: nil,
		},
	},
}

var realPowerCommandBlock = modbus.MetricBlock{
	Name:         "RealPowerCommand",
	StartAddr:    1000,
	NumRegisters: 3,
	Metrics: map[string]modbus.Metric{
		"Mode": {
			StartAddr:   1000,
			DataType:    modbus.Uint16Type,
			ScalingFunc: nil,
		},
		"AlwaysActive": {
			StartAddr:   1001,
			DataType:    modbus.Uint16Type,
			ScalingFunc: nil,
		},
		"PeakPowerMode": {
			StartAddr:   1002,
			DataType:    modbus.Uint16Type,
			ScalingFunc: nil,
		},
	},
}

var statusBlock = modbus.MetricBlock{
	Name:         "Status",
	StartAddr:    200,
	NumRegisters: 34,
	Metrics: map[string]modbus.Metric{

		"CommandSource": {
			StartAddr:   200,
			DataType:    modbus.Uint16Type,
			ScalingFunc: nil,
		},
		"BatteryTargetP": {
			StartAddr:   201,
			DataType:    modbus.Int32Type,
			ScalingFunc: nil,
		},
		"NominalEnergy": {
			StartAddr:   207,
			DataType:    modbus.Int32Type,
			ScalingFunc: nil,
		},
		"AvailableBlocks": {
			StartAddr:   218,
			DataType:    modbus.Uint16Type,
			ScalingFunc: nil,
		},
	},
}

var directRealPowerCommandBlock = modbus.MetricBlock{
	Name:         "DirectRealPowerCommand",
	StartAddr:    1020,
	NumRegisters: 4,
	Metrics: map[string]modbus.Metric{
		"Power": {
			StartAddr:   1020,
			DataType:    modbus.Int32Type,
			ScalingFunc: nil,
		},
		"Heartbeat": {
			StartAddr:   1022,
			DataType:    modbus.Uint16Type,
			ScalingFunc: nil,
		},
		"Timeout": {
			StartAddr:   1023,
			DataType:    modbus.Uint16Type,
			ScalingFunc: nil,
		},
	},
}
