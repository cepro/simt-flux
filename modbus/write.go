package modbus

import (
	"encoding/binary"
	"fmt"
)

// WriteMetric writes the given value to the given modbus metric
func (c *Client) WriteMetric(metric Metric, val interface{}) error {

	err := c.reconnectIfNeccesary()
	if err != nil {
		return fmt.Errorf("reconnect: %w", err)
	}

	bytes := metric.DataType.toBytesFunc(val)
	nBytes := len(bytes)
	registerVals := make([]uint16, 0, nBytes/2)
	for i := 0; i < int(nBytes); i = i + 2 {
		registerVals = append(registerVals, binary.BigEndian.Uint16(bytes[i:i+1]))
	}

	err = c.subClient.WriteRegisters(metric.StartAddr, registerVals)
	if err != nil {
		c.setShouldReconnect()
		return fmt.Errorf("write register %d: %w", metric.StartAddr, err)
	}

	return nil
}
