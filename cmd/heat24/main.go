package main

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/ksinistr/heat24/internal/cache"
	"github.com/ksinistr/heat24/internal/config"
	"github.com/ksinistr/heat24/internal/openmeteo"
	"github.com/ksinistr/heat24/internal/report"
)

const configPath = "config.yaml"

func main() {
	log.SetFlags(0)

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatal(err)
	}
	forecastTTL, err := config.ParseTTL(cfg.Cache.ForecastTTL)
	if err != nil {
		log.Fatalf("cache.forecast_ttl: %v", err)
	}
	archiveTTL, err := config.ParseTTL(cfg.Cache.ArchiveTTL)
	if err != nil {
		log.Fatalf("cache.archive_ttl: %v", err)
	}

	httpClient := retryablehttp.NewClient()
	httpClient.RetryMax = 5
	httpClient.RetryWaitMin = 200 * time.Millisecond
	httpClient.RetryWaitMax = 2 * time.Second
	httpClient.Logger = nil

	source := openmeteo.NewSource(
		openmeteo.NewClient(httpClient.StandardClient()),
		cache.New(cfg.Cache.Dir, time.Now),
		forecastTTL,
		archiveTTL,
	)

	reports := report.NewBuilder(source).Build(cfg, time.Now().Year()-1)
	if err := json.NewEncoder(os.Stdout).Encode(reports); err != nil {
		log.Fatal(err)
	}
}
