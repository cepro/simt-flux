package modo

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sync"
	"time"

	"golang.org/x/exp/slog"
)

const (
	imbalancePriceUrlStr  = "https://api.modoenergy.com/pub/v1/gb/modo/markets/system-price-live"
	imbalanceVolumeUrlStr = "https://api.modoenergy.com/pub/v1/gb/modo/markets/niv-live"
)

// Client communicates with Modo and retrieves the imbalance price and volume predictions
type Client struct {
	client                    http.Client
	lock                      sync.RWMutex   // mutex is used to lock access to `lastImbalancePrice` and `lastImbalancePriceSPTime`, as they may be accessed from different go routines
	lastImbalancePrice        float64        // SSP in p/kWh
	lastImbalancePriceSPTime  time.Time      // Settlement period that the imbalance price relates to
	lastImbalanceVolume       float64        // Imbalance volume in kWh
	lastImbalanceVolumeSPTime time.Time      // Settlement period that the imbalance volume relates to
	londonLocation            *time.Location // Just a cache of the London timezone location so it's not re-created every time
	logger                    *slog.Logger
}

type imbalancePriceResponseItem struct {
	Date              string  `json:"date"`
	SettlementPeriod  int     `json:"settlement_period"`
	PricePoundsPerMwh float64 `json:"system_price"` // Modo returns SSP in £/MWh
}

type imbalancePriceResponse struct {
	Results []imbalancePriceResponseItem `json:"results"`
}

type imbalanceVolumeResponseItem struct {
	Date             string  `json:"date"`
	SettlementPeriod int     `json:"settlement_period"`
	VolumeMwh        float64 `json:"niv"` // Modo returns imbalance volume in MWh
}

type imbalanceVolumeResponse struct {
	Results []imbalanceVolumeResponseItem `json:"results"`
}

func New(client http.Client) *Client {

	londonLocation, err := time.LoadLocation("Europe/London")
	if err != nil {
		panic("Could not load Europe/London location")
	}

	return &Client{
		client:                    client,
		lock:                      sync.RWMutex{},
		lastImbalancePrice:        math.NaN(),
		lastImbalancePriceSPTime:  time.Time{},
		lastImbalanceVolume:       math.NaN(),
		lastImbalanceVolumeSPTime: time.Time{},
		londonLocation:            londonLocation,
		logger:                    slog.Default(),
	}
}

// Run loops forever updating the imbalance price or volume every `period`.
// The calls to get the price and volume are alternated (with a call every `period`) because Modo
// has implemented rate limiting which works across both calls. At the time of writing the rate
// limiting seems to allow 1 call per minute.
func (c *Client) Run(ctx context.Context, period time.Duration) error {
	ticker := time.NewTicker(period)

	processPriceNext := true

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if processPriceNext {
				c.processPrice()
			} else {
				c.processVolume()
			}
			processPriceNext = !processPriceNext
		}
	}
}

func (c *Client) processPrice() {
	c.lock.RLock()
	previousImbalancePrice := c.lastImbalancePrice
	previousImbalancePriceSPTime := c.lastImbalancePriceSPTime
	c.lock.RUnlock()

	err := c.updateImbalancePrice()
	if err != nil {
		c.logger.Error("Failed to update Modo imbalance price", "error", err)
		return
	}

	priceDidChange := (previousImbalancePrice != c.lastImbalancePrice) || !(previousImbalancePriceSPTime.Equal(c.lastImbalancePriceSPTime))
	c.logger.Info(
		"Updated Modo imbalance price",
		"price", c.lastImbalancePrice,
		"price_settlement_perod", c.lastImbalancePriceSPTime,
		"did_change", priceDidChange,
	)
}

func (c *Client) processVolume() {
	c.lock.RLock()
	previousImbalanceVolume := c.lastImbalanceVolume
	previousImbalanceVolumeSPTime := c.lastImbalanceVolumeSPTime
	c.lock.RUnlock()

	err := c.updateImbalanceVolume()
	if err != nil {
		c.logger.Error("Failed to update Modo imbalance volume", "error", err)
		return
	}

	volumeDidChange := (previousImbalanceVolume != c.lastImbalanceVolume) || !(previousImbalanceVolumeSPTime.Equal(c.lastImbalanceVolumeSPTime))
	c.logger.Info(
		"Updated Modo imbalance volume",
		"volume", c.lastImbalanceVolume/1e3,
		"volume_settlement_perod", c.lastImbalanceVolumeSPTime,
		"did_change", volumeDidChange,
	)
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

// requestImbalancePrice returns Modo's latest imbalance price calculation, or an error.
func (c *Client) requestImbalancePrice() (imbalancePriceResponseItem, error) {

	modoUrl, err := url.Parse(imbalancePriceUrlStr)
	if err != nil {
		return imbalancePriceResponseItem{}, err
	}

	dateStr := time.Now().In(c.londonLocation).Format("2006-01-02")

	params := url.Values{}
	params.Add("date_from", dateStr)
	params.Add("date_to", dateStr)
	modoUrl.RawQuery = params.Encode()

	response, err := c.client.Get(modoUrl.String())
	if err != nil {
		return imbalancePriceResponseItem{}, fmt.Errorf("get system price: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return imbalancePriceResponseItem{}, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	parsedResponse := imbalancePriceResponse{}
	err = json.NewDecoder(response.Body).Decode(&parsedResponse)
	if err != nil {
		return imbalancePriceResponseItem{}, fmt.Errorf("parse body: %w", err)
	}

	if len(parsedResponse.Results) < 1 {
		return imbalancePriceResponseItem{}, fmt.Errorf("no results for this day yet")
	}

	latestResult := parsedResponse.Results[0]

	return latestResult, nil
}

// requestImbalanceVolume returns Modo's imbalance price calculation, or an error.
func (c *Client) requestImbalanceVolume() (imbalanceVolumeResponseItem, error) {

	modoUrl, err := url.Parse(imbalanceVolumeUrlStr)
	if err != nil {
		return imbalanceVolumeResponseItem{}, err
	}

	dateStr := time.Now().In(c.londonLocation).Format("2006-01-02")

	params := url.Values{}
	params.Add("date_from", dateStr)
	params.Add("date_to", dateStr)
	modoUrl.RawQuery = params.Encode()

	response, err := c.client.Get(modoUrl.String())
	if err != nil {
		return imbalanceVolumeResponseItem{}, fmt.Errorf("get niv: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return imbalanceVolumeResponseItem{}, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	parsedResponse := imbalanceVolumeResponse{}
	err = json.NewDecoder(response.Body).Decode(&parsedResponse)
	if err != nil {
		return imbalanceVolumeResponseItem{}, fmt.Errorf("parse body: %w", err)
	}

	if len(parsedResponse.Results) < 1 {
		return imbalanceVolumeResponseItem{}, fmt.Errorf("no results for this day yet")
	}

	latestResult := parsedResponse.Results[0]

	return latestResult, nil
}

// timeOfSettlementPeriod returns the start time of the 30min settlement period denoted by the given date and SP number, or an error
func timeOfSettlementPeriod(dateStr string, settlementPeriod int) (time.Time, error) {

	if settlementPeriod < 1 || settlementPeriod > 50 {
		// TODO: we could have further validation of `settlementPeriod` range on clock change days etc
		return time.Time{}, fmt.Errorf("invalid settlement period: %d", settlementPeriod)
	}

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
