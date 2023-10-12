package powerpack

import "github.com/cepro/besscontroller/modbusaccess"

var configBlock = modbusaccess.RegisterBlock{
	Name:         "Config",
	StartAddr:    100,
	NumRegisters: 47,
	Registers: map[string]modbusaccess.Register{

		"ProtocolVersion": {
			StartAddr:   100,
			DataType:    modbusaccess.Int16Type,
			ScalingFunc: nil,
		},
		"FirmwareVersion": {
			StartAddr:   102,
			DataType:    modbusaccess.String32Type,
			ScalingFunc: nil,
		},
		"Serial": {
			StartAddr:   118,
			DataType:    modbusaccess.String32Type,
			ScalingFunc: nil,
		},
		"NumBattMeters": {
			StartAddr:   134,
			DataType:    modbusaccess.Int16Type,
			ScalingFunc: nil,
		},
		"NumSiteMeters": {
			StartAddr:   135,
			DataType:    modbusaccess.Int16Type,
			ScalingFunc: nil,
		},
		"MaxChargePower": {
			StartAddr:   139,
			DataType:    modbusaccess.Int32Type,
			ScalingFunc: nil,
		},
		"MaxDischargePower": {
			StartAddr:   141,
			DataType:    modbusaccess.Int32Type,
			ScalingFunc: nil,
		},
		"Energy": {
			StartAddr:   145,
			DataType:    modbusaccess.Int32Type,
			ScalingFunc: nil,
		},
	},
}

var realPowerCommandBlock = modbusaccess.RegisterBlock{
	Name:         "RealPowerCommand",
	StartAddr:    1000,
	NumRegisters: 3,
	Registers: map[string]modbusaccess.Register{
		"Mode": {
			StartAddr:   1000,
			DataType:    modbusaccess.Uint16Type,
			ScalingFunc: nil,
		},
		"AlwaysActive": {
			StartAddr:   1001,
			DataType:    modbusaccess.Uint16Type,
			ScalingFunc: nil,
		},
		"PeakPowerMode": {
			StartAddr:   1002,
			DataType:    modbusaccess.Uint16Type,
			ScalingFunc: nil,
		},
	},
}

var statusBlock = modbusaccess.RegisterBlock{
	Name:         "Status",
	StartAddr:    200,
	NumRegisters: 34,
	Registers: map[string]modbusaccess.Register{

		"CommandSource": {
			StartAddr:   200,
			DataType:    modbusaccess.Uint16Type,
			ScalingFunc: nil,
		},
		"BatteryTargetP": {
			StartAddr:   201,
			DataType:    modbusaccess.Int32Type,
			ScalingFunc: nil,
		},
		"NominalEnergy": {
			StartAddr:   207,
			DataType:    modbusaccess.Int32Type,
			ScalingFunc: nil,
		},
		"AvailableBlocks": {
			StartAddr:   218,
			DataType:    modbusaccess.Uint16Type,
			ScalingFunc: nil,
		},
	},
}

var directRealPowerCommandBlock = modbusaccess.RegisterBlock{
	Name:         "DirectRealPowerCommand",
	StartAddr:    1020,
	NumRegisters: 4,
	Registers: map[string]modbusaccess.Register{
		"Power": {
			StartAddr:   1020,
			DataType:    modbusaccess.Int32Type,
			ScalingFunc: nil,
		},
		"Heartbeat": {
			StartAddr:   1022,
			DataType:    modbusaccess.Uint16Type,
			ScalingFunc: nil,
		},
		"Timeout": {
			StartAddr:   1023,
			DataType:    modbusaccess.Uint16Type,
			ScalingFunc: nil,
		},
	},
}
