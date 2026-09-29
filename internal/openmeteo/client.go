package openmeteo

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

const (
	forecastURL = "https://api.open-meteo.com/v1/forecast"
	archiveURL  = "https://archive-api.open-meteo.com/v1/archive"
)

type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

type Client struct {
	http Doer
}

func NewClient(http Doer) *Client {
	return &Client{http: http}
}

func (c *Client) LastWeek(lat, lon float64) ([]byte, error) {
	q := baseQuery(lat, lon)
	q.Set("daily", "sunrise,sunset")
	q.Set("past_days", "7")
	q.Set("forecast_days", "0")
	return c.get(forecastURL, q)
}

// start and end are inclusive dates in YYYY-MM-DD form.
func (c *Client) Archive(lat, lon float64, start, end string) ([]byte, error) {
	q := baseQuery(lat, lon)
	q.Set("start_date", start)
	q.Set("end_date", end)
	return c.get(archiveURL, q)
}

func baseQuery(lat, lon float64) url.Values {
	q := url.Values{}
	q.Set("latitude", strconv.FormatFloat(lat, 'f', -1, 64))
	q.Set("longitude", strconv.FormatFloat(lon, 'f', -1, 64))
	q.Set("hourly", "temperature_2m,relative_humidity_2m")
	q.Set("timezone", "auto")
	return q
}

func (c *Client) get(base string, q url.Values) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, base+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("open-meteo: %s: %s", resp.Status, body)
	}
	return body, nil
}
