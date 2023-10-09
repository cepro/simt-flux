package acuvim2

import "github.com/cepro/besscontroller/modbusaccess"

var blocks = []modbusaccess.RegisterBlock{
	{
		Name:         "Power",
		StartAddr:    12288,
		NumRegisters: 60,
		Registers: map[string]modbusaccess.Register{
			"Frequency": {
				StartAddr:   12288,
				DataType:    modbusaccess.FloatType,
				ScalingFunc: nil,
			},
			"VoltageLineAverage": {
				StartAddr:   12304,
				DataType:    modbusaccess.FloatType,
				ScalingFunc: scaleVoltage,
			},
			// Line voltages are available here, but are not of interest at the moment
			"CurrentPhA": {
				StartAddr:   12306,
				DataType:    modbusaccess.FloatType,
				ScalingFunc: scaleCurrent,
			},
			"CurrentPhB": {
				StartAddr:   12308,
				DataType:    modbusaccess.FloatType,
				ScalingFunc: scaleCurrent,
			},
			"CurrentPhC": {
				StartAddr:   12310,
				DataType:    modbusaccess.FloatType,
				ScalingFunc: scaleCurrent,
			},
			"CurrentPhAverage": {
				StartAddr:   12312,
				DataType:    modbusaccess.FloatType,
				ScalingFunc: scaleCurrent,
			},
			// Neutral current is available here, but it's not of interest at the moment
			"PowerPhAActive": {
				StartAddr:   12316,
				DataType:    modbusaccess.FloatType,
				ScalingFunc: scalePower,
			},
			"PowerPhBActive": {
				StartAddr:   12318,
				DataType:    modbusaccess.FloatType,
				ScalingFunc: scalePower,
			},
			"PowerPhCActive": {
				StartAddr:   12320,
				DataType:    modbusaccess.FloatType,
				ScalingFunc: scalePower,
			},
			"PowerTotalActive": {
				StartAddr:   12322,
				DataType:    modbusaccess.FloatType,
				ScalingFunc: scalePower,
			},
			// Reactive power by phase is available here, but it's not of interest at the moment
			"PowerTotalReactive": {
				StartAddr:   12330,
				DataType:    modbusaccess.FloatType,
				ScalingFunc: scalePower,
			},
			// Apparent power by phase is available here, but it's not of interest at the moment
			"PowerTotalApparent": {
				StartAddr:   12338,
				DataType:    modbusaccess.FloatType,
				ScalingFunc: scalePower,
			},
			// Power factor by phase is available here, but it's not of interest at the moment
			"PowerFactorTotal": {
				StartAddr:   12346,
				DataType:    modbusaccess.FloatType,
				ScalingFunc: nil,
			},
		},
	},
	{
		Name:         "Energy",
		StartAddr:    16456,
		NumRegisters: 8,
		Registers: map[string]modbusaccess.Register{
			"EnergyImportedActive": {
				StartAddr:   16456,
				DataType:    modbusaccess.Int32Type,
				ScalingFunc: scaleEnergy,
			},
			"EnergyExportedActive": {
				StartAddr:   16458,
				DataType:    modbusaccess.Int32Type,
				ScalingFunc: scaleEnergy,
			},
			"EnergyImportedReactive": {
				StartAddr:   16460,
				DataType:    modbusaccess.Int32Type,
				ScalingFunc: scaleEnergy,
			},
			"EnergyExportedReactive": {
				StartAddr:   16462,
				DataType:    modbusaccess.Int32Type,
				ScalingFunc: scaleEnergy,
			},
		},
	},
}

func scaleVoltage(scaler modbusaccess.Scaler, val interface{}) interface{} {
	meter := scaler.(*Acuvim2Meter)
	return val.(float64) * (meter.pt1 / meter.pt2)
}

func scaleCurrent(scaler modbusaccess.Scaler, val interface{}) interface{} {
	meter := scaler.(*Acuvim2Meter)
	return val.(float64) * (meter.ct1 / meter.ct2)
}

func scalePower(scaler modbusaccess.Scaler, val interface{}) interface{} {
	meter := scaler.(*Acuvim2Meter)
	return (val.(float64) * (meter.pt1 / meter.pt2) * (meter.ct1 / meter.ct2)) / 1000
}

func scaleEnergy(scaler modbusaccess.Scaler, val interface{}) interface{} {
	return val.(int32) / 10
}
