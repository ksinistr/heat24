package report

import (
	"log"
	"math"
	"strconv"
	"time"

	"github.com/nsr888/heat24/internal/config"
	"github.com/nsr888/heat24/internal/heat"
	"github.com/nsr888/heat24/internal/weather"
)

type Source interface {
	LastWeek(name string, lat, lon float64) (*weather.Response, error)
	Month(name string, lat, lon float64, year int, month time.Month) (*weather.Response, error)
	Year(name string, lat, lon float64, year int) (*weather.Response, error)
}

// Value encodes NaN as JSON null.
type Value float64

func (v Value) MarshalJSON() ([]byte, error) {
	f := float64(v)
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return []byte("null"), nil
	}
	return strconv.AppendFloat(nil, math.Round(f*100)/100, 'f', -1, 64), nil
}

type Reports struct {
	LastWeek []LastWeekChart `json:"lastWeek"`
	Month    []MonthChart    `json:"month"`
	Annual   AnnualReport    `json:"annual"`
}

type LastWeekChart struct {
	Location   string  `json:"location"`
	Error      string  `json:"error,omitempty"`
	Timezone   string  `json:"timezone,omitempty"`
	Hourly     []Value `json:"hourly,omitempty"`
	SunriseMin int     `json:"sunriseMin,omitempty"`
	SunsetMin  int     `json:"sunsetMin,omitempty"`
}

type MonthSeries struct {
	Location string  `json:"location"`
	Hourly   []Value `json:"hourly"`
}

type MonthFailure struct {
	Location string `json:"location"`
	Error    string `json:"error"`
}

type MonthChart struct {
	Year     int            `json:"year"`
	Month    int            `json:"month"`
	Series   []MonthSeries  `json:"series"`
	Failures []MonthFailure `json:"failures,omitempty"`
}

type AnnualReport struct {
	Year   int           `json:"year"`
	Charts []AnnualChart `json:"charts"`
}

type AnnualChart struct {
	Location string     `json:"location"`
	Error    string     `json:"error,omitempty"`
	Grid     [][]Value  `json:"grid,omitempty"`
}

type Builder struct {
	source Source
}

func NewBuilder(source Source) *Builder {
	return &Builder{source: source}
}

func (b *Builder) Build(cfg config.Config, year int) Reports {
	return Reports{
		LastWeek: b.lastWeek(cfg.Reports.LastWeek.Locations),
		Month:    b.month(cfg.Reports.Month, year),
		Annual:   b.annual(cfg.Reports.Annual.Locations, year),
	}
}

func (b *Builder) lastWeek(locations []config.Location) []LastWeekChart {
	charts := make([]LastWeekChart, 0, len(locations))
	for _, l := range locations {
		chart, err := b.lastWeekChart(l)
		if err != nil {
			log.Printf("last week: %s: %v", l.Name, err)
			chart = LastWeekChart{Location: l.Name, Error: err.Error()}
		}
		charts = append(charts, chart)
	}
	return charts
}

func (b *Builder) lastWeekChart(l config.Location) (LastWeekChart, error) {
	r, err := b.source.LastWeek(l.Name, l.Lat, l.Lon)
	if err != nil {
		return LastWeekChart{}, err
	}
	tz, err := r.Location()
	if err != nil {
		return LastWeekChart{}, err
	}
	samples, err := r.Samples(tz)
	if err != nil {
		return LastWeekChart{}, err
	}
	sunrise, sunset, err := r.MedianSunTimes(tz)
	if err != nil {
		return LastWeekChart{}, err
	}
	hourly := heat.HourlyHybrid(samples)
	return LastWeekChart{
		Location:   l.Name,
		Timezone:   tz.String(),
		Hourly:     values(hourly[:]),
		SunriseMin: sunrise,
		SunsetMin:  sunset,
	}, nil
}

func (b *Builder) month(cfg config.Month, year int) []MonthChart {
	charts := make([]MonthChart, 0, len(cfg.Months))
	for _, m := range cfg.Months {
		chart := MonthChart{Year: year, Month: m, Series: []MonthSeries{}}
		for _, l := range cfg.Locations {
			hourly, err := b.monthHourly(l, year, time.Month(m))
			if err != nil {
				log.Printf("month %04d-%02d: %s: %v", year, m, l.Name, err)
				chart.Failures = append(chart.Failures, MonthFailure{Location: l.Name, Error: err.Error()})
				continue
			}
			chart.Series = append(chart.Series, MonthSeries{Location: l.Name, Hourly: values(hourly[:])})
		}
		charts = append(charts, chart)
	}
	return charts
}

func (b *Builder) monthHourly(l config.Location, year int, month time.Month) ([24]float64, error) {
	r, err := b.source.Month(l.Name, l.Lat, l.Lon, year, month)
	if err != nil {
		return [24]float64{}, err
	}
	tz, err := r.Location()
	if err != nil {
		return [24]float64{}, err
	}
	samples, err := r.Samples(tz)
	if err != nil {
		return [24]float64{}, err
	}
	return heat.HourlyHybrid(samples), nil
}

func (b *Builder) annual(locations []config.Location, year int) AnnualReport {
	out := AnnualReport{Year: year, Charts: make([]AnnualChart, 0, len(locations))}
	for _, l := range locations {
		grid, err := b.annualGrid(l, year)
		if err != nil {
			log.Printf("annual %04d: %s: %v", year, l.Name, err)
			out.Charts = append(out.Charts, AnnualChart{Location: l.Name, Error: err.Error()})
			continue
		}
		out.Charts = append(out.Charts, AnnualChart{Location: l.Name, Grid: grid})
	}
	return out
}

func (b *Builder) annualGrid(l config.Location, year int) ([][]Value, error) {
	r, err := b.source.Year(l.Name, l.Lat, l.Lon, year)
	if err != nil {
		return nil, err
	}
	tz, err := r.Location()
	if err != nil {
		return nil, err
	}
	samples, err := r.Samples(tz)
	if err != nil {
		return nil, err
	}
	grid := heat.HourMonthGrid(samples)
	rows := make([][]Value, 24)
	for h := range 24 {
		rows[h] = values(grid[h][:])
	}
	return rows, nil
}

func values(in []float64) []Value {
	out := make([]Value, len(in))
	for i, v := range in {
		out[i] = Value(v)
	}
	return out
}
