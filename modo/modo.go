package modo

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"golang.org/x/exp/slog"
)

const (
	systemPriceUrl = "https://data-api.modoenergy.com/v1/widgets/system-price/"
)

type Client struct {
	client                   http.Client
	lock                     sync.RWMutex // mutex is used to lock access to `lastImbalancePrice` and `lastImbalancePriceSPTime`
	lastImbalancePrice       float64      // SSP in p/kWh
	lastImbalancePriceSPTime time.Time
	logger                   *slog.Logger
}

type systemPriceResponse struct {
	Date              string  `json:"date"`
	SettlementPeriod  int     `json:"settlement_period"`
	PricePoundsPerMwh float64 `json:"system_price"` // Modo returns SSP in £/MWh
}

func New(client http.Client) *Client {
	return &Client{
		client:                   client,
		lock:                     sync.RWMutex{},
		lastImbalancePrice:       math.NaN(),
		lastImbalancePriceSPTime: time.Time{},
		logger:                   slog.Default(),
	}
}

// Run loops forever updating the imbalance price every `period`
func (c *Client) Run(ctx context.Context, period time.Duration) error {
	ticker := time.NewTicker(period)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:

			// TODO: we should have a read lock activated here
			previousImbalancePrice := c.lastImbalancePrice
			previousImbalancePriceSPTime := c.lastImbalancePriceSPTime

			err := c.updateImbalancePrice()
			if err != nil {
				c.logger.Error("Failed to update Modo imbalance price", "error", err)
				continue
			}

			// TODO: didChange doesn't work
			didChange := (previousImbalancePrice != c.lastImbalancePrice) || !(previousImbalancePriceSPTime.Equal(c.lastImbalancePriceSPTime))

			c.logger.Info("Updated Modo imbalance price", "settlement_perod", c.lastImbalancePriceSPTime, "imbalance_price", c.lastImbalancePrice, "did_change", didChange)
		}
	}
}

// ImbalancePrice returns the last cached imbalance price, and the settlement period time that it corresponds to
func (c *Client) ImbalancePrice() (float64, time.Time) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.lastImbalancePrice, c.lastImbalancePriceSPTime
}

// updateImbalancePrice updates the cached imbalance price by querying Modo's servers.
func (c *Client) updateImbalancePrice() error {
	parsedResponse, err := c.requestSystemPrice()
	if err != nil {
		return err
	}

	t, err := timeOfSettlementPeriod(parsedResponse.Date, parsedResponse.SettlementPeriod)
	if err != nil {
		return fmt.Errorf("parse settlement period: %w", err)
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	c.lastImbalancePrice = parsedResponse.PricePoundsPerMwh / 10
	c.lastImbalancePriceSPTime = t

	return nil
}

// requestSystemPrice returns Modo's imbalance price calculation, or an error.
func (c *Client) requestSystemPrice() (systemPriceResponse, error) {
	response, err := c.client.Get(systemPriceUrl)
	if err != nil {
		return systemPriceResponse{}, fmt.Errorf("get system price: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return systemPriceResponse{}, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	parsedResponse := systemPriceResponse{}
	err = json.NewDecoder(response.Body).Decode(&parsedResponse)
	if err != nil {
		return systemPriceResponse{}, fmt.Errorf("parse body: %w", err)
	}

	return parsedResponse, nil
}

// timeOfSettlementPeriod returns the start time of the 30min settlement period denoted by the given date and SP number, or an error
func timeOfSettlementPeriod(dateStr string, settlementPeriod int) (time.Time, error) {

	if settlementPeriod < 1 || settlementPeriod > 50 {
		return time.Time{}, fmt.Errorf("invalid settlement period: %d", settlementPeriod)
	}
	// TODO: we could have further validation of `settlementPeriod` range on clock change days etc

	// Go doesn't have built-in libraries to support standalone Dates (it's all Datetimes). So we first parse the date string (into a datetime)
	// and then later re-create the datetime with timezone etc.
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse date: %w", err)
	}

	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		return time.Time{}, fmt.Errorf("load london tz: %w", err)
	}

	t := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, london)
	t = t.Add(time.Duration(settlementPeriod-1) * time.Duration(time.Minute*30))

	return t, nil
}
