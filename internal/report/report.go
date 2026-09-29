package report

import (
	"log"
	"math"
	"strconv"
	"time"

	"github.com/ksinistr/heat24/internal/config"
	"github.com/ksinistr/heat24/internal/heat"
	"github.com/ksinistr/heat24/internal/openmeteo"
)

type Source interface {
	LastWeek(p openmeteo.Point) (openmeteo.LastWeek, error)
	Year(p openmeteo.Point, year int) ([]heat.Sample, error)
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
	RiskLevels []RiskLevel     `json:"riskLevels"`
	LastWeek   []LastWeekChart `json:"lastWeek"`
	Month      []MonthChart    `json:"month"`
	Annual     AnnualReport    `json:"annual"`
}

type RiskLevel struct {
	Name string `json:"name"`
	Max  Value  `json:"max"`
}

type LastWeekChart struct {
	Location   string  `json:"location"`
	Error      string  `json:"error,omitempty"`
	Timezone   string  `json:"timezone,omitempty"`
	From       string  `json:"from,omitempty"`
	To         string  `json:"to,omitempty"`
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
	Year        int            `json:"year"`
	Month       int            `json:"month"`
	Series      []MonthSeries  `json:"series"`
	Comfortable []string       `json:"comfortable"`
	Failures    []MonthFailure `json:"failures,omitempty"`
}

type AnnualReport struct {
	Year   int           `json:"year"`
	Charts []AnnualChart `json:"charts"`
}

type AnnualChart struct {
	Location string    `json:"location"`
	Error    string    `json:"error,omitempty"`
	Grid     [][]Value `json:"grid,omitempty"`
}

type Builder struct {
	source Source
}

func NewBuilder(source Source) *Builder {
	return &Builder{source: source}
}

func (b *Builder) Build(cfg config.Config, year int) Reports {
	return Reports{
		RiskLevels: riskLevels(),
		LastWeek:   b.lastWeek(cfg.Reports.LastWeek.Locations),
		Month:      b.month(cfg.Reports.Month, year),
		Annual:     b.annual(cfg.Reports.Annual.Locations, year),
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
	w, err := b.source.LastWeek(l.Point)
	if err != nil {
		return LastWeekChart{}, err
	}
	hourly := heat.HourlyHybrid(w.Samples)
	from, to := dateRange(w.Samples)
	return LastWeekChart{
		Location:   l.Name,
		Timezone:   w.Timezone,
		From:       from,
		To:         to,
		Hourly:     values(hourly[:]),
		SunriseMin: w.SunriseMin,
		SunsetMin:  w.SunsetMin,
	}, nil
}

func (b *Builder) month(cfg config.Month, year int) []MonthChart {
	samples := make([][]heat.Sample, len(cfg.Locations))
	errs := make([]error, len(cfg.Locations))
	for i, l := range cfg.Locations {
		samples[i], errs[i] = b.source.Year(l.Point, year)
		if errs[i] != nil {
			log.Printf("month %04d: %s: %v", year, l.Name, errs[i])
		}
	}
	charts := make([]MonthChart, 0, len(cfg.Months))
	for _, m := range cfg.Months {
		chart := MonthChart{Year: year, Month: m, Series: []MonthSeries{}, Comfortable: []string{}}
		for i, l := range cfg.Locations {
			if errs[i] != nil {
				chart.Failures = append(chart.Failures, MonthFailure{Location: l.Name, Error: errs[i].Error()})
				continue
			}
			hourly := heat.HourlyHybrid(heat.InMonth(samples[i], time.Month(m)))
			chart.Series = append(chart.Series, MonthSeries{Location: l.Name, Hourly: values(hourly[:])})
			if heat.Comfortable(hourly) {
				chart.Comfortable = append(chart.Comfortable, l.Name)
			}
		}
		charts = append(charts, chart)
	}
	return charts
}

func (b *Builder) annual(locations []config.Location, year int) AnnualReport {
	out := AnnualReport{Year: year, Charts: make([]AnnualChart, 0, len(locations))}
	for _, l := range locations {
		samples, err := b.source.Year(l.Point, year)
		if err != nil {
			log.Printf("annual %04d: %s: %v", year, l.Name, err)
			out.Charts = append(out.Charts, AnnualChart{Location: l.Name, Error: err.Error()})
			continue
		}
		out.Charts = append(out.Charts, AnnualChart{Location: l.Name, Grid: grid(heat.HourMonthGrid(samples))})
	}
	return out
}

func dateRange(samples []heat.Sample) (from, to string) {
	if len(samples) == 0 {
		return "", ""
	}
	first, last := samples[0].Time, samples[0].Time
	for _, s := range samples[1:] {
		if s.Time.Before(first) {
			first = s.Time
		}
		if s.Time.After(last) {
			last = s.Time
		}
	}
	return first.Format(time.DateOnly), last.Format(time.DateOnly)
}

func riskLevels() []RiskLevel {
	levels := heat.RiskLevels()
	out := make([]RiskLevel, len(levels))
	for i, l := range levels {
		out[i] = RiskLevel{Name: l.Name, Max: Value(l.MaxC)}
	}
	return out
}

func grid(in [24][12]float64) [][]Value {
	rows := make([][]Value, 24)
	for h := range 24 {
		rows[h] = values(in[h][:])
	}
	return rows
}

func values(in []float64) []Value {
	out := make([]Value, len(in))
	for i, v := range in {
		out[i] = Value(v)
	}
	return out
}
