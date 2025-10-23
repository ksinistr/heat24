package internal

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/hashicorp/go-retryablehttp"
)

type openMeteoResp struct {
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	Timezone       string  `json:"timezone"`
	TimezoneAbbrev string  `json:"timezone_abbreviation"`

	Hourly struct {
		Time               []string  `json:"time"`
		Temperature2m      []float64 `json:"temperature_2m"`
		RelativeHumidity2m []float64 `json:"relative_humidity_2m"`
	} `json:"hourly"`

	Daily struct {
		Time    []string `json:"time"`
		Sunrise []string `json:"sunrise"`
		Sunset  []string `json:"sunset"`
	} `json:"daily"`
}
type WeatherFetcher struct {
	client *retryablehttp.Client
}

func NewWeatherFetcher(cacheDir string) *WeatherFetcher {
	rc := retryablehttp.NewClient()
	rc.RetryMax = 5
	rc.RetryWaitMin = 200 * time.Millisecond
	rc.RetryWaitMax = 2 * time.Second

	// On-disk HTTP cache to avoid verbose bespoke caching.
	cache := diskcache.New(cacheDir)
	rc.HTTPClient.Transport = httpcache.NewTransport(cache)

	return &WeatherFetcher{client: rc}
}

func (wf *WeatherFetcher) Fetch(lat, lon float64) (*openMeteoResp, error) {
	url := fmt.Sprintf(
		"https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&hourly=temperature_2m,relative_humidity_2m&daily=sunrise,sunset&past_days=7&forecast_days=0&timezone=auto",
		lat, lon,
	)
	req, err := retryablehttp.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := wf.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var om openMeteoResp
	if err := json.NewDecoder(resp.Body).Decode(&om); err != nil {
		return nil, err
	}
	return &om, nil
}
