package weather

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nsr888/heat24/internal/heat"
)

type Point struct {
	Lat float64 `yaml:"lat"`
	Lon float64 `yaml:"lon"`
}

func (p Point) key() string {
	return fmt.Sprintf("%.4f_%.4f", p.Lat, p.Lon)
}

type LastWeek struct {
	Timezone   string
	Samples    []heat.Sample
	SunriseMin int
	SunsetMin  int
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

func (s *Source) LastWeek(p Point) (LastWeek, error) {
	key := fmt.Sprintf("forecast/%s.json", p.key())
	r, err := s.load(key, s.forecastTTL, func() ([]byte, error) {
		return s.client.LastWeek(p.Lat, p.Lon)
	})
	if err != nil {
		return LastWeek{}, err
	}
	tz, err := time.LoadLocation(r.Timezone)
	if err != nil {
		return LastWeek{}, err
	}
	samples, err := r.samples(tz)
	if err != nil {
		return LastWeek{}, err
	}
	sunrise, sunset, err := r.medianSunTimes(tz)
	if err != nil {
		return LastWeek{}, err
	}
	return LastWeek{Timezone: tz.String(), Samples: samples, SunriseMin: sunrise, SunsetMin: sunset}, nil
}

func (s *Source) Year(p Point, year int) ([]heat.Sample, error) {
	key := fmt.Sprintf("archive/%s-%04d.json", p.key(), year)
	r, err := s.load(key, s.archiveTTL, func() ([]byte, error) {
		return s.client.Archive(p.Lat, p.Lon, fmt.Sprintf("%04d-01-01", year), fmt.Sprintf("%04d-12-31", year))
	})
	if err != nil {
		return nil, err
	}
	tz, err := time.LoadLocation(r.Timezone)
	if err != nil {
		return nil, err
	}
	return r.samples(tz)
}

func (s *Source) load(key string, ttl time.Duration, fetch func() ([]byte, error)) (*response, error) {
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
	var r response
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("%s: %w", key, err)
	}
	return &r, nil
}
