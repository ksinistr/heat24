package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

type openMeteoResp struct {
	Timezone string `json:"timezone"`
	Hourly   struct {
		Time               []string  `json:"time"`
		Temperature2m      []float64 `json:"temperature_2m"`
		RelativeHumidity2m []float64 `json:"relative_humidity_2m"`
	} `json:"hourly"`
}

type HeatMapData struct {
	sums   [12][24]float64
	counts [12][24]int
}

func (hmd *HeatMapData) add(mo, hr int, hi float64) {
	if !math.IsNaN(hi) {
		hmd.sums[mo][hr] += hi
		hmd.counts[mo][hr]++
	}
}

func (hmd *HeatMapData) avg(mo, hr int) float64 {
	if hmd.counts[mo][hr] == 0 {
		return math.NaN()
	}
	return hmd.sums[mo][hr] / float64(hmd.counts[mo][hr])
}

func heatIndexC(tempC, rh float64) float64 {
	if tempC < 27.0 || rh < 40.0 {
		return tempC
	}
	T := tempC*9.0/5.0 + 32.0
	R := rh

	HI := -42.379 +
		2.04901523*T +
		10.14333127*R -
		0.22475541*T*R -
		6.83783e-3*T*T -
		5.481717e-2*R*R +
		1.22874e-3*T*T*R +
		8.5282e-4*T*R*R -
		1.99e-6*T*T*R*R

	if R < 13 && T >= 80 && T <= 112 {
		HI -= ((13 - R) / 4.0) * math.Sqrt((17.0-math.Abs(T-95.0))/17.0)
	}
	if R > 85 && T >= 80 && T <= 87 {
		HI += ((R - 85.0) / 10.0) * ((87.0 - T) / 5.0)
	}

	return (HI - 32.0) * 5.0 / 9.0
}

func fetchArchive(lat, lon float64) (*openMeteoResp, error) {
	u := url.URL{
		Scheme: "https",
		Host:   "archive-api.open-meteo.com",
		Path:   "/v1/archive",
	}
	q := u.Query()
	q.Set("latitude", fmt.Sprintf("%f", lat))
	q.Set("longitude", fmt.Sprintf("%f", lon))
	q.Set("start_date", "2024-01-01")
	q.Set("end_date", "2024-12-31")
	q.Set("hourly", "temperature_2m,relative_humidity_2m")
	q.Set("timezone", "auto")
	u.RawQuery = q.Encode()

	client := retryablehttp.NewClient()
	client.RetryMax = 3
	resp, err := client.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var om openMeteoResp
	if err := json.NewDecoder(resp.Body).Decode(&om); err != nil {
		return nil, err
	}

	return &om, nil
}

func processData(om *openMeteoResp) *HeatMapData {
	loc, err := time.LoadLocation(om.Timezone)
	if err != nil {
		log.Fatal(err)
	}

	hmd := &HeatMapData{}

	n := len(om.Hourly.Time)
	for i := 0; i < n; i++ {
		t, err := time.ParseInLocation("2006-01-02T15:04", om.Hourly.Time[i], loc)
		if err != nil {
			log.Printf("parse time %s: %v", om.Hourly.Time[i], err)
			continue
		}

		mo := int(t.Month()) - 1 // 0-11
		hr := t.Hour()           // 0-23

		temp := om.Hourly.Temperature2m[i]
		rh := om.Hourly.RelativeHumidity2m[i]

		hi := heatIndexC(temp, rh)

		hmd.add(mo, hr, hi)
	}

	return hmd
}

type offsetUnitGrid struct {
	XOffset, YOffset float64
	Data             *mat.Dense
}

func (g offsetUnitGrid) Dims() (c, r int)   { r, c = g.Data.Dims(); return c, r }
func (g offsetUnitGrid) Z(c, r int) float64 { return g.Data.At(r, c) }
func (g offsetUnitGrid) X(c int) float64    { return float64(c) + g.XOffset }
func (g offsetUnitGrid) Y(r int) float64    { return float64(r) + g.YOffset }

var (
	LightGreen = color.RGBA{144, 238, 144, 255}
	Yellow     = color.RGBA{255, 255, 0, 255}
	Orange     = color.RGBA{255, 165, 0, 255}
	Red        = color.RGBA{255, 0, 0, 255}
	DarkRed    = color.RGBA{139, 0, 0, 255}
)

type DiscretePalette struct {
	colors   []color.Color
	binEdges []float64
}

func (p *DiscretePalette) Colors() []color.Color {
	return p.colors
}

func (p *DiscretePalette) Len() int {
	return len(p.colors)
}

func NewDiscretePalette() *DiscretePalette {
	pal := &DiscretePalette{
		colors:   []color.Color{LightGreen, Yellow, Orange, Red, DarkRed},
		binEdges: []float64{27, 32, 41, 54},
	}
	return pal
}

func (p *DiscretePalette) ColorAt(z float64) color.Color {
	if math.IsNaN(z) || z < 27 {
		return LightGreen
	}
	for i, edge := range p.binEdges {
		if z < edge {
			return p.colors[i+1]
		}
	}
	return DarkRed
}

func (p *DiscretePalette) Min() float64 {
	return 20
}

func (p *DiscretePalette) Max() float64 {
	return 60
}

// Month tick markers.
type monthTicks struct {
	Names []string
}

func (m monthTicks) Ticks(min, max float64) []plot.Tick {
	var ticks []plot.Tick
	for i := int(math.Ceil(min)); i <= int(math.Floor(max)); i++ {
		if i < 0 || i >= len(m.Names) {
			continue
		}
		ticks = append(ticks, plot.Tick{Value: float64(i), Label: m.Names[i]})
	}
	return ticks
}

// Hour tick markers (skip some for readability).
type hourTicks struct {
	Step int
}

func (h hourTicks) Ticks(min, max float64) []plot.Tick {
	var ticks []plot.Tick
	for i := int(math.Ceil(min)); i <= int(math.Floor(max)); i++ {
		if i%h.Step == 0 {
			ticks = append(ticks, plot.Tick{Value: float64(i), Label: fmt.Sprintf("%02d:00", i)})
		} else {
			ticks = append(ticks, plot.Tick{Value: float64(i)})
		}
	}
	return ticks
}

// Simple thumbnail implementation for legend
type colorThumbnail struct {
	color color.Color
}

func (ct *colorThumbnail) Thumbnail(c *draw.Canvas) {
	pts := []vg.Point{
		{X: c.Min.X, Y: c.Min.Y},
		{X: c.Min.X, Y: c.Max.Y},
		{X: c.Max.X, Y: c.Max.Y},
		{X: c.Max.X, Y: c.Min.Y},
	}
	poly := c.ClipPolygonY(pts)
	c.FillPolygon(ct.color, poly)
}

type Location struct {
	Name string
	Lat  float64
	Lon  float64
}

var locations = []Location{
	{"Paphos", 34.7768, 32.4245},
	// {"Novi Sad", 45.2517, 19.8369},
	// {"Palemi", 34.88593, 32.50657},
	// {"Pana Panagia", 34.91901721271778, 32.630531461579665},
	// {"Limmasol", 34.7071, 33.0226},
	// {"Nicosia", 35.1856, 33.3823},
	// {"Larnaca", 34.9190, 33.6232},
	// {"Famagusta", 35.1264, 33.9197},
	// {"Troodos", 34.9886, 32.8662},
}

func main() {
	const hours = 24
	const months = 12

	for _, loc := range locations {
		lat := loc.Lat
		lon := loc.Lon

		om, err := fetchArchive(lat, lon)
		if err != nil {
			log.Fatalf("fetch: %v", err)
		}

		hmd := processData(om)

		data := make([]float64, hours*months)
		for mo := range months {
			for hr := range hours {
				data[hr*months+mo] = hmd.avg(mo, hr)
			}
		}

		matData := mat.NewDense(hours, months, data)
		grid := offsetUnitGrid{
			XOffset: 0,
			YOffset: 0,
			Data:    matData,
		}

		pal := NewDiscretePalette()
		hm := plotter.NewHeatMap(grid, pal)

		p := plot.New()
		p.Title.Text = fmt.Sprintf("Annual Heat Index Heatmap (Hour vs Month) - %s", loc.Name)
		p.X.Label.Text = "Month"
		p.Y.Label.Text = "Hour of Day"

		p.X.Tick.Marker = monthTicks{Names: []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}}
		p.Y.Tick.Marker = hourTicks{Step: 2}

		p.X.Min = 0
		p.X.Max = float64(months)
		p.Y.Min = 0
		p.Y.Max = float64(hours)

		p.X.Padding = 0
		p.Y.Padding = 0

		p.Add(hm)

		// Legend for discrete colors.
		l := plot.NewLegend()
		legendEntries := []struct {
			Label string
			Color color.Color
		}{
			{"<27°C", LightGreen},
			{"27-32°C", Yellow},
			{"32-41°C", Orange},
			{"41-54°C", Red},
			{">54°C", DarkRed},
		}

		// Add legend entries with custom thumbnails
		for _, entry := range legendEntries {
			thumb := &colorThumbnail{color: entry.Color}
			l.Add(entry.Label, thumb)
		}

		l.Top = true

		img := vgimg.New(800, 600)
		dc := draw.New(img)

		// Legend layout.
		r := l.Rectangle(dc)
		legendWidth := r.Max.X - r.Min.X
		l.Draw(dc)
		dc = draw.Crop(dc, 0, -legendWidth-vg.Length(5*vg.Millimeter), 0, 0)
		p.Draw(dc)

		safeName := strings.ReplaceAll(strings.ToLower(loc.Name), " ", "_")
		filename := fmt.Sprintf("heat_index_annual_%s.png", safeName)
		out, err := os.Create(filename)
		if err != nil {
			log.Fatal(err)
		}
		defer out.Close()
		png := vgimg.PngCanvas{Canvas: img}
		if _, err = png.WriteTo(out); err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Saved %s\n", filename)
	}
}
