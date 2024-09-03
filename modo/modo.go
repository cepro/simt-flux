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
	imbalancePriceUrl  = "https://admin.modo.energy/v1/data-api/widgets/system-price/"
	imbalanceVolumeUrl = "https://admin.modo.energy/v1/data-api/widgets/system-imbalance/"
)

type Client struct {
	client                    http.Client
	lock                      sync.RWMutex // mutex is used to lock access to `lastImbalancePrice` and `lastImbalancePriceSPTime`
	lastImbalancePrice        float64      // SSP in p/kWh
	lastImbalancePriceSPTime  time.Time    // Settlement period that the imbalance price relates to
	lastImbalanceVolume       float64      // Imbalance volume in kWh
	lastImbalanceVolumeSPTime time.Time    // Settlement period that the imbalance volume relates to
	logger                    *slog.Logger
}

type imbalancePriceResponse struct {
	Date              string  `json:"date"`
	SettlementPeriod  int     `json:"settlement_period"`
	PricePoundsPerMwh float64 `json:"system_price"` // Modo returns SSP in £/MWh
}

type imbalanceVolumeResponse struct {
	Date             string  `json:"date"`
	SettlementPeriod int     `json:"settlement_period"`
	VolumeMwh        float64 `json:"niv"` // Modo returns imbalance volume in MWh
}

func New(client http.Client) *Client {
	return &Client{
		client:                    client,
		lock:                      sync.RWMutex{},
		lastImbalancePrice:        math.NaN(),
		lastImbalancePriceSPTime:  time.Time{},
		lastImbalanceVolume:       math.NaN(),
		lastImbalanceVolumeSPTime: time.Time{},
		logger:                    slog.Default(),
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
			previousImbalanceVolume := c.lastImbalanceVolume
			previousImbalanceVolumeSPTime := c.lastImbalanceVolumeSPTime

			err := c.updateImbalancePrice()
			if err != nil {
				c.logger.Error("Failed to update Modo imbalance price", "error", err)
				continue
			}

			err = c.updateImbalanceVolume()
			if err != nil {
				c.logger.Error("Failed to update Modo imbalance volume", "error", err)
				continue
			}

			priceDidChange := (previousImbalancePrice != c.lastImbalancePrice) || !(previousImbalancePriceSPTime.Equal(c.lastImbalancePriceSPTime))
			volumeDidChange := (previousImbalanceVolume != c.lastImbalanceVolume) || !(previousImbalanceVolumeSPTime.Equal(c.lastImbalanceVolumeSPTime))
			c.logger.Info(
				"Updated Modo imbalance price and volume",
				"price", c.lastImbalancePrice,
				"price_settlement_perod", c.lastImbalancePriceSPTime,
				"volume", c.lastImbalanceVolume/1e3,
				"volume_settlement_perod", c.lastImbalanceVolumeSPTime,
				"did_change", priceDidChange || volumeDidChange,
			)

		}
	}
}

// ImbalancePrice returns the last cached imbalance price, and the settlement period time that it corresponds to
func (c *Client) ImbalancePrice() (float64, time.Time) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.lastImbalancePrice, c.lastImbalancePriceSPTime
}

// ImbalanceVolume returns the last cached imbalance volume, and the settlement period time that it corresponds to
func (c *Client) ImbalanceVolume() (float64, time.Time) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.lastImbalanceVolume, c.lastImbalanceVolumeSPTime
}

// updateImbalancePrice updates the cached imbalance price by querying Modo's servers.
func (c *Client) updateImbalancePrice() error {
	parsedResponse, err := c.requestImbalancePrice()
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

// updateImbalanceVolume updates the cached imbalance volume by querying Modo's servers.
func (c *Client) updateImbalanceVolume() error {
	parsedResponse, err := c.requestImbalanceVolume()
	if err != nil {
		return err
	}

	t, err := timeOfSettlementPeriod(parsedResponse.Date, parsedResponse.SettlementPeriod)
	if err != nil {
		return fmt.Errorf("parse settlement period: %w", err)
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	c.lastImbalanceVolume = parsedResponse.VolumeMwh * 1e3
	c.lastImbalanceVolumeSPTime = t

	return nil
}

// requestImbalancePrice returns Modo's imbalance price calculation, or an error.
func (c *Client) requestImbalancePrice() (imbalancePriceResponse, error) {
	response, err := c.client.Get(imbalancePriceUrl)
	if err != nil {
		return imbalancePriceResponse{}, fmt.Errorf("get system price: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return imbalancePriceResponse{}, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	parsedResponse := imbalancePriceResponse{}
	err = json.NewDecoder(response.Body).Decode(&parsedResponse)
	if err != nil {
		return imbalancePriceResponse{}, fmt.Errorf("parse body: %w", err)
	}

	return parsedResponse, nil
}

// requestImbalanceVolume returns Modo's imbalance price calculation, or an error.
func (c *Client) requestImbalanceVolume() (imbalanceVolumeResponse, error) {
	response, err := c.client.Get(imbalanceVolumeUrl)
	if err != nil {
		return imbalanceVolumeResponse{}, fmt.Errorf("get system imbalance: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return imbalanceVolumeResponse{}, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	parsedResponse := imbalanceVolumeResponse{}
	err = json.NewDecoder(response.Body).Decode(&parsedResponse)
	if err != nil {
		return imbalanceVolumeResponse{}, fmt.Errorf("parse body: %w", err)
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
