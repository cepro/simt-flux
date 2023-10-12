package modbusaccess

import (
	"fmt"
	"maps"

	"github.com/grid-x/modbus"
)

// PollBlocks reads all the register `blocks` from the `client` and returns a map of the parsed values, keyed by metric name.
// The `scaler` instance is passed into any scaling functions defined in the register block.
func PollBlocks(client modbus.Client, scaler Scaler, blocks []RegisterBlock) (map[string]interface{}, error) {

	allMetrics := make(map[string]interface{})

	for _, block := range blocks {
		blockMetrics, err := PollBlock(client, scaler, block)
		if err != nil {
			return nil, fmt.Errorf("poll block '%s': %w", block.Name, err)
		}
		maps.Copy(allMetrics, blockMetrics)
	}

	return allMetrics, nil
}

// pollBlock reads a single register `block` from the `client` and returns a map of the parsed values, keyed by metric name.
// The `scaler` instance is passed into any scaling functions defined in the register block.
func PollBlock(client modbus.Client, scaler Scaler, block RegisterBlock) (map[string]interface{}, error) {

	// read the whole block of bytes from the modbus device
	bytes, err := client.ReadHoldingRegisters(block.StartAddr, block.NumRegisters)
	if err != nil {
		return nil, fmt.Errorf("read block: %w", err)
	}

	// extract each metric of interest from the block of bytes
	metrics := make(map[string]interface{}, len(block.Registers))
	for key, register := range block.Registers {

		// sanity check the configuration to avoid out of bound panics
		offset := (int(register.StartAddr) - int(block.StartAddr)) * 2 // registers are two bytes long
		if offset < 0 {
			return nil, fmt.Errorf("register configuration for `%s` preceeds block", key)
		}
		if offset+int(register.DataType.dataLength) > len(bytes) {
			return nil, fmt.Errorf("register configuration for '%s' exceeds block", key)
		}

		// grab the relevant bytes for this metric from the block of bytes
		registerBytes := bytes[offset:(offset + int(register.DataType.dataLength))]

		// convert the bytes into the concrete data type (mostly these are floats)
		val := register.DataType.fromBytesFunc(registerBytes)

		// scale the value as required by the products modbus specification
		if register.ScalingFunc != nil {
			val = register.ScalingFunc(scaler, val)
		}

		metrics[key] = val
	}

	return metrics, nil
}
