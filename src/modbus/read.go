package modbus

import (
	"encoding/binary"
	"fmt"
	"maps"

	"github.com/simonvetter/modbus"
)

// PollBlocks reads all the metric `blocks` from the `client` and returns a map of the parsed values, keyed by metric name.
// The `scaler` instance is passed into any scaling functions defined in the register block.
func (c *Client) PollBlocks(scaler Scaler, blocks []MetricBlock) (map[string]interface{}, error) {

	allMetricVals := make(map[string]interface{})

	for _, block := range blocks {
		blockMetricVals, err := c.PollBlock(scaler, block)
		if err != nil {
			return nil, fmt.Errorf("poll block '%s': %w", block.Name, err)
		}
		maps.Copy(allMetricVals, blockMetricVals)
	}

	return allMetricVals, nil
}

// pollBlock reads a single metric `block` from the `client` and returns a map of the parsed values, keyed by metric name.
// The `scaler` instance is passed into any scaling functions defined in the register block.
func (c *Client) PollBlock(scaler Scaler, block MetricBlock) (map[string]interface{}, error) {

	err := c.reconnectIfNeccesary()
	if err != nil {
		return nil, fmt.Errorf("reconnect: %w", err)
	}

	// read the whole block of bytes from the modbus device
	registerVals, err := c.subClient.ReadRegisters(block.StartAddr, block.NumRegisters, modbus.HOLDING_REGISTER)
	if err != nil {
		c.setShouldReconnect()
		return nil, fmt.Errorf("read block: %w", err)
	}

	// Each register is a uint16, convert into a byte array
	bytes := make([]byte, len(registerVals)*2)
	for i, registerVal := range registerVals {
		loc := i * 2
		binary.BigEndian.PutUint16(bytes[loc:loc+2], registerVal)
	}

	// extract each metric of interest from the block of bytes
	metricVals := make(map[string]interface{}, len(block.Metrics))
	for key, register := range block.Metrics {

		// sanity check the modbus register configuration to avoid out of bound panics
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
		metricVal := register.DataType.fromBytesFunc(registerBytes)

		// scale the value as required by the products modbus specification
		if register.ScalingFunc != nil {
			metricVal = register.ScalingFunc(scaler, metricVal)
		}

		metricVals[key] = metricVal
	}

	return metricVals, nil
}
