package supabase

import (
	"errors"
	"fmt"
	"time"

	supa "github.com/nedpals/supabase-go"
	"golang.org/x/exp/slog"
)

const (
	supabaseUploadTimeout = time.Second * 10
)

// Client provides an interface onto the Supabase platform.
// It hides the underlying open source supabase library and adds reconnection and timeout logic.
type Client struct {
	url     string
	anonKey string
	userKey string
	schema  string

	subClient       *supa.Client // the raw client of the underlying supabase library we are using
	shouldReconnect bool         // when true, the subClient is 'dirty' and will be re-created next time a read or write call is made
	logger          *slog.Logger
}

func New(url, anonKey, userKey, schema string) (*Client, error) {
	client := &Client{
		url:             url,
		anonKey:         anonKey,
		userKey:         userKey,
		schema:          schema,
		shouldReconnect: true, // shouldReconnect is marked as true from instantiation so the connection will be made lazily on the first request to read or write
		logger:          slog.Default().With("host", url),
	}

	return client, nil
}

// UploadReadings takes the given readings of any type, and attempts to upload to the relevant supabase table.
func (c *Client) UploadReadings(readings interface{}) error {

	c.reconnectIfNeccesary()

	// The supabase client library doesn't have good timeout support, so here we wrap the call in a timeout
	errCh := make(chan error, 1)
	go func() {
		// Convert the 'original readings' (e.g. telemetry.BessReading) into the supabase types (e.g. supabaseBessReading)
		supabaseReadings, supabaseTableName := convertReadingsForSupabase(readings)
		errCh <- c.subClient.DB.From(supabaseTableName).Insert(supabaseReadings).Execute(nil)
	}()

	select {
	case <-time.After(supabaseUploadTimeout):
		c.setShouldReconnect()
		return errors.New("timed out")
	case err := <-errCh:
		if err != nil {
			c.setShouldReconnect()
		}
		return err
	}
}

// createSubClient creates the open-source supabase library client with sensible defaults and connects to the host.
func (c *Client) createSubClient() error {

	subClient := supa.CreateClient(c.url, c.anonKey)

	// The supabase client library doesn't have a fully featured interface, here we specify options directly by
	// adding headers to the postgrest requests.
	// Use the appropriate schema:
	subClient.DB.AddHeader("Accept-Profile", c.schema)
	subClient.DB.AddHeader("Content-Profile", c.schema)

	// Use a user JWT:
	if c.userKey != "" {
		subClient.DB.AddHeader("Authorization", fmt.Sprintf("Bearer %s", c.userKey))
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

	err := c.createSubClient()
	if err != nil {
		return err
	}

	c.shouldReconnect = false

	c.logger.Info("Created supabase client")

	return nil
}
