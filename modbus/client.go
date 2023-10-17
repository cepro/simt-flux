package modbus

import (
	"fmt"
	"time"

	"github.com/simonvetter/modbus"
	"golang.org/x/exp/slog"
)

// Client provides an interface onto Modbus devices.
// It hides the underlying open source modbus library and provides functionality to map metrics to their assigned registers.
type Client struct {
	host string

	subClient       *modbus.ModbusClient // the raw client of the underlying modbus library we are using
	shouldReconnect bool                 // when true, the subClient is 'dirty' and will be re-created
	logger          *slog.Logger
}

func NewClient(host string) (*Client, error) {
	client := &Client{
		host:            host,
		shouldReconnect: false,
		logger:          slog.Default().With("host", host),
	}

	err := client.createSubClient()
	if err != nil {
		return nil, err
	}

	return client, nil
}

// createSubClient creates the open-source modbus library client with sensible defaults and connects to the host.
func (c *Client) createSubClient() error {
	subClient, err := modbus.NewClient(&modbus.ClientConfiguration{
		URL:     fmt.Sprintf("tcp://%s", c.host),
		Timeout: 2 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("create modbus client: %w", err)
	}

	err = subClient.Open()
	if err != nil {
		return fmt.Errorf("open modbus client: %w", err)
	}

	c.subClient = subClient

	return nil
}

// setShouldReconnect is called when there has been an error with the modbus connection that should trigger a re-connect.
func (c *Client) setShouldReconnect() {
	c.shouldReconnect = true
}

// reconnectIfNeccesary will close the old connection and reconnect if there have been problems with the connection.
func (c *Client) reconnectIfNeccesary() error {
	if !c.shouldReconnect {
		return nil
	}

	// Ignore errors from Close() as we will continue with the reconnect anyway and start a new connection.
	c.subClient.Close()

	err := c.createSubClient()
	if err != nil {
		return err
	}

	c.shouldReconnect = false

	c.logger.Info("Reconnected modbus client")

	return nil
}
