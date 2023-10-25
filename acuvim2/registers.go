package acuvim2

import "github.com/cepro/besscontroller/modbus"

var blocks = []modbus.MetricBlock{
	{
		Name:         "Power",
		StartAddr:    12288,
		NumRegisters: 60,
		Metrics: map[string]modbus.Metric{
			"Frequency": {
				StartAddr:   12288,
				DataType:    modbus.FloatType,
				ScalingFunc: nil,
			},
			"VoltageLineAverage": {
				StartAddr:   12304,
				DataType:    modbus.FloatType,
				ScalingFunc: scaleVoltage,
			},
			// Line voltages are available here, but are not of interest at the moment
			"CurrentPhA": {
				StartAddr:   12306,
				DataType:    modbus.FloatType,
				ScalingFunc: scaleCurrent,
			},
			"CurrentPhB": {
				StartAddr:   12308,
				DataType:    modbus.FloatType,
				ScalingFunc: scaleCurrent,
			},
			"CurrentPhC": {
				StartAddr:   12310,
				DataType:    modbus.FloatType,
				ScalingFunc: scaleCurrent,
			},
			"CurrentPhAverage": {
				StartAddr:   12312,
				DataType:    modbus.FloatType,
				ScalingFunc: scaleCurrent,
			},
			// Neutral current is available here, but it's not of interest at the moment
			"PowerPhAActive": {
				StartAddr:   12316,
				DataType:    modbus.FloatType,
				ScalingFunc: scalePower,
			},
			"PowerPhBActive": {
				StartAddr:   12318,
				DataType:    modbus.FloatType,
				ScalingFunc: scalePower,
			},
			"PowerPhCActive": {
				StartAddr:   12320,
				DataType:    modbus.FloatType,
				ScalingFunc: scalePower,
			},
			"PowerTotalActive": {
				StartAddr:   12322,
				DataType:    modbus.FloatType,
				ScalingFunc: scalePower,
			},
			// Reactive power by phase is available here, but it's not of interest at the moment
			"PowerTotalReactive": {
				StartAddr:   12330,
				DataType:    modbus.FloatType,
				ScalingFunc: scalePower,
			},
			// Apparent power by phase is available here, but it's not of interest at the moment
			"PowerTotalApparent": {
				StartAddr:   12338,
				DataType:    modbus.FloatType,
				ScalingFunc: scalePower,
			},
			// Power factor by phase is available here, but it's not of interest at the moment
			"PowerFactorTotal": {
				StartAddr:   12346,
				DataType:    modbus.FloatType,
				ScalingFunc: nil,
			},
		},
	},
	{
		Name:         "Energy",
		StartAddr:    16456,
		NumRegisters: 8,
		Metrics: map[string]modbus.Metric{
			"EnergyImportedActive": {
				StartAddr:   16456,
				DataType:    modbus.Int32Type,
				ScalingFunc: scaleEnergy,
			},
			"EnergyExportedActive": {
				StartAddr:   16458,
				DataType:    modbus.Int32Type,
				ScalingFunc: scaleEnergy,
			},
			"EnergyImportedReactive": {
				StartAddr:   16460,
				DataType:    modbus.Int32Type,
				ScalingFunc: scaleEnergy,
			},
			"EnergyExportedReactive": {
				StartAddr:   16462,
				DataType:    modbus.Int32Type,
				ScalingFunc: scaleEnergy,
			},
		},
	},
	{
		Name:         "EnergyPerPhase",
		StartAddr:    17952,
		NumRegisters: 12,
		Metrics: map[string]modbus.Metric{
			"EnergyImportedPhAActive": {
				StartAddr:   17952,
				DataType:    modbus.Int32Type,
				ScalingFunc: scaleEnergy,
			},
			"EnergyExportedPhAActive": {
				StartAddr:   17954,
				DataType:    modbus.Int32Type,
				ScalingFunc: scaleEnergy,
			},
			"EnergyImportedPhBActive": {
				StartAddr:   17956,
				DataType:    modbus.Int32Type,
				ScalingFunc: scaleEnergy,
			},
			"EnergyExportedPhBActive": {
				StartAddr:   17958,
				DataType:    modbus.Int32Type,
				ScalingFunc: scaleEnergy,
			},
			"EnergyImportedPhCActive": {
				StartAddr:   17960,
				DataType:    modbus.Int32Type,
				ScalingFunc: scaleEnergy,
			},
			"EnergyExportedPhCActive": {
				StartAddr:   17962,
				DataType:    modbus.Int32Type,
				ScalingFunc: scaleEnergy,
			},
		},
	},
}

func scaleVoltage(scaler modbus.Scaler, val interface{}) interface{} {
	meter := scaler.(*Acuvim2Meter)
	return val.(float64) * (meter.pt1 / meter.pt2)
}

func scaleCurrent(scaler modbus.Scaler, val interface{}) interface{} {
	meter := scaler.(*Acuvim2Meter)
	return val.(float64) * (meter.ct1 / meter.ct2)
}

func scalePower(scaler modbus.Scaler, val interface{}) interface{} {
	meter := scaler.(*Acuvim2Meter)
	return (val.(float64) * (meter.pt1 / meter.pt2) * (meter.ct1 / meter.ct2)) / 1000
}

func scaleEnergy(scaler modbus.Scaler, val interface{}) interface{} {
	return val.(int32) / 10
}
