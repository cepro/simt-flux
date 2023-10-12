package modbusaccess

import (
	"bytes"
	"encoding/binary"
	"math"
)

// Type represents the different types of data that can be queried over modbus.
type Type struct {
	name          string                   // the name of the data type
	dataLength    uint16                   // the number of underlying bytes to represent the data type
	fromBytesFunc func([]byte) interface{} // function to convert the bytes to the concrete data type (used to read from modbus)
	toBytesFunc   func(interface{}) []byte // function to convert the concrete data type into bytes (used to write to modbus)
}

// FloatType represents the float data type.
var FloatType = Type{
	name:       "float",
	dataLength: 4,
	fromBytesFunc: func(bytes []byte) interface{} {
		valUint32 := binary.BigEndian.Uint32(bytes)
		valFloat32 := math.Float32frombits(valUint32)
		return float64(valFloat32)
	},
	toBytesFunc: nil,
}

// Int32Type represents the 32 bit signed integer data type on Modbus.
var Int32Type = Type{
	name:       "int32",
	dataLength: 4,
	fromBytesFunc: func(bytes []byte) interface{} {
		valUint32 := binary.BigEndian.Uint32(bytes)
		valInt32 := int32(valUint32)
		return valInt32
	},
	toBytesFunc: func(val interface{}) []byte {
		bytes := make([]byte, 4)
		binary.BigEndian.PutUint32(bytes, val.(uint32))
		return bytes
	},
}

// Uint16Type represents the 16 bit unsigned integer data type on Modbus.
var Uint16Type = Type{
	name:       "uint16",
	dataLength: 2,
	fromBytesFunc: func(bytes []byte) interface{} {
		valUint16 := binary.BigEndian.Uint16(bytes)
		return valUint16
	},
	toBytesFunc: func(val interface{}) []byte {
		bytes := make([]byte, 2)
		binary.BigEndian.PutUint16(bytes, val.(uint16))
		return bytes
	},
}

// Int16Type represents the 16 bit signed integer data type on Modbus.
var Int16Type = Type{
	name:       "int16",
	dataLength: 2,
	fromBytesFunc: func(bytes []byte) interface{} {
		valUint16 := binary.BigEndian.Uint16(bytes)
		valInt16 := int16(valUint16)
		return valInt16
	},
	toBytesFunc: nil,
}

// String32Type represents a 32 byte long, null terminated string
var String32Type = Type{
	name:       "string32",
	dataLength: 32,
	fromBytesFunc: func(b []byte) interface{} {
		return string(bytes.Trim(b, "\x00"))
	},
	toBytesFunc: nil,
}

// Scaler can be any object used to help scale modbus values.
// For trivial scaling scenarios (e.g. 'divide by 1000') this is not really required, but for more complicated scaling
// scenarios (e.g. 'scale by the configured current transformer ratios') it can be neccesary to retrieve state from the `scaler`.
type Scaler interface{}

// valueScalingFunc is a prototype for a function that scales a modbus value.
type valueScalingFunc func(Scaler, interface{}) interface{}

// Register holds a value on the modbus slave at the given address
type Register struct {
	StartAddr   uint16
	DataType    Type
	ScalingFunc valueScalingFunc // a function to scale the recieved value to get it's 'true' value (transmitting scaled values is common in Modbus)
}

// RegisterBlock represents a contigous block of modbus registers that are read in one chunk.
type RegisterBlock struct {
	Name         string              // name of the block used for context/logging
	StartAddr    uint16              // the first register address of the block
	NumRegisters uint16              // the number of registers in this block (each register is two bytes)
	Registers    map[string]Register // details of all the registers of interest in this block, keyed by unique name
}
