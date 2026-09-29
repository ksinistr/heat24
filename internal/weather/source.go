package weather

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Response struct {
	Timezone string `json:"timezone"`
	Hourly   struct {
		Time               []string   `json:"time"`
		Temperature2m      []*float64 `json:"temperature_2m"`
		RelativeHumidity2m []*float64 `json:"relative_humidity_2m"`
	} `json:"hourly"`
	Daily struct {
		Sunrise []string `json:"sunrise"`
		Sunset  []string `json:"sunset"`
	} `json:"daily"`
}

type Cache interface {
	Get(key string, ttl time.Duration) ([]byte, bool, error)
	Put(key string, data []byte) error
}

type Source struct {
	client      *Client
	cache       Cache
	forecastTTL time.Duration
	archiveTTL  time.Duration
}

func NewSource(client *Client, cache Cache, forecastTTL, archiveTTL time.Duration) *Source {
	return &Source{client: client, cache: cache, forecastTTL: forecastTTL, archiveTTL: archiveTTL}
}

func (s *Source) LastWeek(name string, lat, lon float64) (*Response, error) {
	key := fmt.Sprintf("last_week/%s.json", slug(name))
	return s.load(key, s.forecastTTL, func() ([]byte, error) {
		return s.client.LastWeek(lat, lon)
	})
}

func (s *Source) Month(name string, lat, lon float64, year int, month time.Month) (*Response, error) {
	first := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	last := first.AddDate(0, 1, -1)
	key := fmt.Sprintf("month/%s-%04d-%02d.json", slug(name), year, int(month))
	return s.load(key, s.archiveTTL, func() ([]byte, error) {
		return s.client.Archive(lat, lon, first.Format(time.DateOnly), last.Format(time.DateOnly))
	})
}

func (s *Source) Year(name string, lat, lon float64, year int) (*Response, error) {
	key := fmt.Sprintf("annual/%s-%04d.json", slug(name), year)
	return s.load(key, s.archiveTTL, func() ([]byte, error) {
		return s.client.Archive(lat, lon, fmt.Sprintf("%04d-01-01", year), fmt.Sprintf("%04d-12-31", year))
	})
}

func (s *Source) load(key string, ttl time.Duration, fetch func() ([]byte, error)) (*Response, error) {
	data, ok, err := s.cache.Get(key, ttl)
	if err != nil {
		return nil, err
	}
	if !ok {
		data, err = fetch()
		if err != nil {
			return nil, err
		}
		if err := s.cache.Put(key, data); err != nil {
			return nil, err
		}
	}
	var r Response
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("%s: %w", key, err)
	}
	return &r, nil
}

func slug(name string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(name)), " ", "_")
}
