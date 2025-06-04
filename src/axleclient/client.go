package axleclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	accessTokenMaxAge = time.Second * 20 // how old an Axle access token can be before we get a new one
)

// Client implements the API onto the Axle cloud.
type Client struct {
	httpClient http.Client
	baseUrl    string
	username   string
	password   string

	accessToken            string
	accessTokenLastUpdated time.Time

	logger *slog.Logger
}

// authResponse is the JSON body that is sent by Axle when we query the `auth/token-form` endpoint
type authResponse struct {
	AccessToken string `json:"access_token"`
}

func New(httpClient http.Client, baseUrl, username, password string) *Client {
	client := &Client{
		httpClient: httpClient,
		baseUrl:    baseUrl,
		username:   username,
		password:   password,
		logger:     slog.Default().With("host", baseUrl),
	}

	return client
}

// GetSchedule pulls the latest schedule for the given asset ID from Axle and returns it.
func (c *Client) GetSchedule(assetId string) (Schedule, error) {

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/entities/asset/%s/battery-schedule", c.baseUrl, assetId),
		nil,
	)
	if err != nil {
		return Schedule{}, err
	}

	err = c.authorizeRequest(req)
	if err != nil {
		return Schedule{}, fmt.Errorf("authorization: %w", err)
	}

	response, err := c.httpClient.Do(req)
	if err != nil {
		return Schedule{}, fmt.Errorf("get schedules: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return Schedule{}, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	parsedResponse := Schedule{}
	err = json.NewDecoder(response.Body).Decode(&parsedResponse)
	if err != nil {
		return Schedule{}, fmt.Errorf("parse body: %w", err)
	}

	parsedResponse.ReceivedTime = time.Now()

	return parsedResponse, nil
}

// Sends the given readings/telemetry to the Axle cloud
func (c *Client) UploadReadings(axleReadings []Reading) error {

	readingsData, err := json.Marshal(axleReadings)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/data/readings", c.baseUrl),
		bytes.NewBuffer(readingsData),
	)
	if err != nil {
		return err
	}

	err = c.authorizeRequest(req)
	if err != nil {
		return fmt.Errorf("authorization: %w", err)
	}

	response, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("post readings: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	for _, reading := range axleReadings {
		slog.Info("Uploaded Axle reading", "label", reading.Label, "value", reading.Value, "time", reading.StartTimestamp, "status_code", response.StatusCode)
	}

	return nil
}

// authorizeRequest adds the required Authorization header with access token to the given request (updating the access token as required).
func (c *Client) authorizeRequest(req *http.Request) error {

	if (time.Since(c.accessTokenLastUpdated)) >= accessTokenMaxAge {
		err := c.updateAccessToken()
		if err != nil {
			return fmt.Errorf("update access token: %w", err)
		}
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))

	return nil
}

// updateAccessToken queries the Axle auth endpoint for a new access token and saves it
func (c *Client) updateAccessToken() error {

	// The body of the request uses url encoding
	data := url.Values{}
	data.Set("username", c.username)
	data.Set("password", c.password)

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/auth/token-form", c.baseUrl),
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("get auth: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	parsedResponse := authResponse{}
	err = json.NewDecoder(response.Body).Decode(&parsedResponse)
	if err != nil {
		return fmt.Errorf("parse body: %w", err)
	}

	c.accessToken = parsedResponse.AccessToken
	c.accessTokenLastUpdated = time.Now()

	return nil
}
