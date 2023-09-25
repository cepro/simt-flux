package powerpack

import "github.com/cepro/besscontroller/modbusaccess"

var blocks = []modbusaccess.RegisterBlock{
	{
		Name:         "Config",
		StartAddr:    100,
		NumRegisters: 47,
		Registers: map[string]modbusaccess.Register{

			"ProtocolVersion": {
				StartAddr:   100,
				DataType:    modbusaccess.Int16Type,
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

			// "NominalEnergy": {
			// 	StartAddr:   208,
			// 	DataType:    modbusaccess.Int32Type,
			// 	ScalingFunc: nil,
			// },
		},
	},
}
