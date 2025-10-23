package main

// SRP-focused rewrite using gonum/plot APIs that exist in the docs.
// - Fetcher: Open-Meteo client with retry + disk HTTP cache
// - Analyzer: per-hour means & Heat Index
// - Planner: recommended activity windows
// - Plotter: saves one PDF per location using plot.Save

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/nsr888/heat24/internal"
)

type Location struct {
	Name string
	Lat  float64
	Lon  float64
}

var locations = []Location{
	{"Paphos", 34.7768, 32.4245},
	// {"Palemi", 34.88593, 32.50657},
	// {"Pana Panagia", 34.91901721271778, 32.630531461579665},
	// {"Limmasol", 34.7071, 33.0226},
	// {"Nicosia", 35.1856, 33.3823},
	// {"Larnaca", 34.9190, 33.6232},
	// {"Famagusta", 35.1264, 33.9197},
	// {"Troodos", 34.9886, 32.8662},
}

func main() {
	log.SetFlags(0)

	cacheDir := ".httpcache"
	_ = os.MkdirAll(cacheDir, 0o755)
	fetcher := internal.NewWeatherFetcher(cacheDir)
	pl := internal.NewPlotter()

	outDir := "plots"
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatal(err)
	}

	for _, loc := range locations {
		om, err := fetcher.Fetch(loc.Lat, loc.Lon)
		if err != nil {
			log.Fatalf("fetch %s: %v", loc.Name, err)
		}
		avg, tz, err := internal.PerHourMeans(om)
		if err != nil {
			log.Fatalf("analyze %s: %v", loc.Name, err)
		}
		hi := internal.HourHeatIndex(avg)
		sunriseMin, sunsetMin, err := internal.MedianSunTimes(om, tz)
		if err != nil {
			log.Fatalf("sun times %s: %v", loc.Name, err)
		}
		win := internal.PlanWindows(hi, sunriseMin, sunsetMin)

		fmt.Printf("%s — Sunrise %02d:%02d, Sunset %02d:%02d (%s)\n",
			loc.Name, sunriseMin/60, sunriseMin%60, sunsetMin/60, sunsetMin%60, tz)

		safeName := strings.ReplaceAll(strings.ToLower(loc.Name), " ", "_")
		file := filepath.Join(outDir, fmt.Sprintf("heat_index_%s.pdf", safeName))
		if err := pl.PlotToPDF(file, loc.Name, tz, hi, sunriseMin, sunsetMin, win); err != nil {
			log.Fatalf("plot %s: %v", loc.Name, err)
		}
	}

	fmt.Printf("\nSaved %d PDF plots in %s/\n", len(locations), outDir)
}
