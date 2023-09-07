package acuvim2

import (
	"encoding/binary"
	"math"
)

// modbusDataType represents the different types of data that can be queried over modbus.
type modbusDataType struct {
	name            string                   // the name of the data type
	dataLength      uint16                   // the number of underlying bytes to represent the data type
	bytesToTypeFunc func([]byte) interface{} // function to convert the bytes to the concrete data type
}

// floatModbusDataType represents the float data type.
var floatModbusDataType = modbusDataType{
	name:            "float",
	dataLength:      4,
	bytesToTypeFunc: byteToFloat,
}

// valueScalingFunc is a prototype for a function that scales a modbus value
type valueScalingFunc func(*Acuvim2Meter, interface{}) interface{}

// modbusRegister holds a value on the modbus slave at the given address
type modbusRegister struct {
	startAddr   uint16
	dataType    modbusDataType
	scalingFunc valueScalingFunc // a function to scale the recieved value to get it's 'true' value (transmitting scaled values is common in Modbus)
}

// modbusRegisterBlock represents a contigous block of modbus registers that are read in one chunk.
type modbusRegisterBlock struct {
	startAddr    uint16                    // the first register address of the block
	numRegisters uint16                    // the number of registers in this block (each register is two bytes)
	registers    map[string]modbusRegister // details of all the registers of interest in this block, keyed by unique name
}

var powerBlock = modbusRegisterBlock{
	startAddr:    12288,
	numRegisters: 60,
	registers: map[string]modbusRegister{
		"Frequency": {
			startAddr:   12288,
			dataType:    floatModbusDataType,
			scalingFunc: nil,
		},
		"VoltageLineAverage": {
			startAddr:   12304,
			dataType:    floatModbusDataType,
			scalingFunc: scaleVoltage,
		},
		// Line voltages are available here, but are not of interest at the moment
		"CurrentPhA": {
			startAddr:   12306,
			dataType:    floatModbusDataType,
			scalingFunc: scaleCurrent,
		},
		"CurrentPhB": {
			startAddr:   12308,
			dataType:    floatModbusDataType,
			scalingFunc: scaleCurrent,
		},
		"CurrentPhC": {
			startAddr:   12310,
			dataType:    floatModbusDataType,
			scalingFunc: scaleCurrent,
		},
		"CurrentPhAverage": {
			startAddr:   12312,
			dataType:    floatModbusDataType,
			scalingFunc: scaleCurrent,
		},
		// Neutral current is available here, but it's not of interest at the moment
		"PowerPhAActive": {
			startAddr:   12316,
			dataType:    floatModbusDataType,
			scalingFunc: scalePower,
		},
		"PowerPhBActive": {
			startAddr:   12318,
			dataType:    floatModbusDataType,
			scalingFunc: scalePower,
		},
		"PowerPhCActive": {
			startAddr:   12320,
			dataType:    floatModbusDataType,
			scalingFunc: scalePower,
		},
		"PowerTotalActive": {
			startAddr:   12322,
			dataType:    floatModbusDataType,
			scalingFunc: scalePower,
		},
		// Reactive power by phase is available here, but it's not of interest at the moment
		"PowerTotalReactive": {
			startAddr:   12330,
			dataType:    floatModbusDataType,
			scalingFunc: scalePower,
		},
		// Apparent power by phase is available here, but it's not of interest at the moment
		"PowerTotalApparent": {
			startAddr:   12338,
			dataType:    floatModbusDataType,
			scalingFunc: scalePower,
		},
		// Power factor by phase is available here, but it's not of interest at the moment
		"PowerFactorTotal": {
			startAddr:   12346,
			dataType:    floatModbusDataType,
			scalingFunc: nil,
		},
	},
}

var energyBlock = modbusRegisterBlock{
	startAddr:    16456,
	numRegisters: 4,
	registers: map[string]modbusRegister{
		"EnergyImported": {
			startAddr:   16456,
			dataType:    floatModbusDataType,
			scalingFunc: scaleEnergy,
		},
		"EnergyExported": {
			startAddr:   16458,
			dataType:    floatModbusDataType,
			scalingFunc: scaleEnergy,
		},
	},
}

// byteToFloat converts 4 bytes into a float value (using big endian)
func byteToFloat(bytes []byte) interface{} {
	valUint32 := binary.BigEndian.Uint32(bytes)
	valFloat32 := math.Float32frombits(valUint32)
	return float64(valFloat32)
}

func scaleVoltage(a *Acuvim2Meter, val interface{}) interface{} {
	return val.(float64) * (a.pt1 / a.pt2)
}

func scaleCurrent(a *Acuvim2Meter, val interface{}) interface{} {
	return val.(float64) * (a.ct1 / a.ct2)
}

func scalePower(a *Acuvim2Meter, val interface{}) interface{} {
	return (val.(float64) * (a.pt1 / a.pt2) * (a.ct1 / a.ct2)) / 1000
}

func scaleEnergy(a *Acuvim2Meter, val interface{}) interface{} {
	return val.(float64) / 1000
}
