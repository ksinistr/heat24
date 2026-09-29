# heat24

A Go tool that visualises daily heat stress by hour for a set of locations, to pick safe times for outdoor activity. It fetches hourly weather from [Open-Meteo](https://open-meteo.com/), computes a heat index, and renders three reports in a static web page.

## Reports

- **Last Week**: per-location 24h curve of the Hybrid Index averaged over the previous 7 days, filled by risk level, with sunrise/sunset.
- **Month**: for each configured month of the previous calendar year, one 24h chart overlaying every location, with risk level bands and a list of comfortable locations.
- **Annual**: per-location heatmap of the Hybrid Index by hour of day × month over the previous calendar year.

## Heat model

- **Heat Index**: NOAA apparent temperature (Rothfusz regression) from air temperature and relative humidity.
- **Hybrid Index**: air temperature below 27 °C, Heat Index at or above 27 °C. This is the value every report displays.
- **Risk levels** (Hybrid Index): Suitable (<27 °C), Caution (27–32), Extreme Caution (32–41), Danger (41–54), Extreme Danger (≥54).
- **Comfortable location**: a location with data where every hour is Suitable. A location that failed to load is never comfortable.

See [CONTEXT.md](CONTEXT.md) for the full glossary.

## Requirements

- Go >= 1.23
- `xdg-open` (only for `make web`)

## Usage

```bash
make web
```

This runs `go run ./cmd/heat24`, which prints all reports as JSON to stdout, writes them to `web/data.js` as `window.HEAT_DATA = ...`, and opens `web/index.html` in the browser.

To get only the JSON:

```bash
go run ./cmd/heat24 > reports.json
```

The program reads `config.yaml` from the current directory. Locations that fail to load are logged to stderr and reported with an error in the output; they do not abort the run.
