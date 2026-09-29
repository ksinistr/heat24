package config

import (
	"fmt"
	"time"

	"github.com/jinzhu/configor"
	"github.com/nsr888/heat24/internal/weather"
)

type Location struct {
	Name          string `yaml:"name"`
	weather.Point `yaml:",inline"`
}

type Cache struct {
	Dir         string `yaml:"dir"`
	ForecastTTL string `yaml:"forecast_ttl"`
	ArchiveTTL  string `yaml:"archive_ttl"`
}

type LastWeek struct {
	Locations []Location `yaml:"locations"`
}

type Month struct {
	Months    []int      `yaml:"months"`
	Locations []Location `yaml:"locations"`
}

type Annual struct {
	Locations []Location `yaml:"locations"`
}

type Config struct {
	Cache   Cache `yaml:"cache"`
	Reports struct {
		LastWeek LastWeek `yaml:"last_week"`
		Month    Month    `yaml:"month"`
		Annual   Annual   `yaml:"annual"`
	} `yaml:"reports"`
}

func Load(path string) (Config, error) {
	var cfg Config
	if err := configor.New(&configor.Config{ErrorOnUnmatchedKeys: true}).Load(&cfg, path); err != nil {
		return Config{}, err
	}
	for _, m := range cfg.Reports.Month.Months {
		if m < 1 || m > 12 {
			return Config{}, fmt.Errorf("reports.month.months: invalid month %d", m)
		}
	}
	return cfg, nil
}

func ParseTTL(s string) (time.Duration, error) {
	if s == "" || s == "0" {
		return 0, nil
	}
	return time.ParseDuration(s)
}
