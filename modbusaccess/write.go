package modbusaccess

import (
	"fmt"

	"github.com/grid-x/modbus"
)

// WriteRegister writes the given value to the given modbus register
func WriteRegister(client modbus.Client, register Register, val interface{}) error {

	bytes := register.DataType.toBytesFunc(val)
	_, err := client.WriteMultipleRegisters(register.StartAddr, register.DataType.dataLength/2, bytes)
	if err != nil {
		return fmt.Errorf("write register %d: %w", register.StartAddr, err)
	}

	return nil
}
