package internal

import (
	"fmt"
	"math"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

type Plotter struct{}

func NewPlotter() *Plotter { return &Plotter{} }

func (p *Plotter) PlotToPDF(filename, locName string, tz *time.Location, hi []float64, sunriseMin, sunsetMin int, win Windows) error {
	plt := plot.New()
	plt.Title.Text = fmt.Sprintf("24h Heat Index (7-day hourly averages) - %s", locName)
	plt.X.Label.Text = "Hour"
	plt.Y.Label.Text = "Heat Index (°C)"

	// X as 0..23 hours
	pts := make(plotter.XYs, 24)
	yMin, yMax := math.Inf(1), math.Inf(-1)
	for h := range 24 {
		pts[h].X = float64(h)
		pts[h].Y = hi[h]
		if !math.IsNaN(hi[h]) {
			if hi[h] < yMin {
				yMin = hi[h]
			}
			if hi[h] > yMax {
				yMax = hi[h]
			}
		}
	}
	// pad y-range a bit
	if !math.IsInf(yMin, 1) && !math.IsInf(yMax, -1) {
		margin := 1.0
		plt.Y.Min = yMin - margin
		plt.Y.Max = yMax + margin
	}

	// Heat index line
	line, err := plotter.NewLine(pts)
	if err != nil {
		return err
	}
	plt.Add(line)
	plt.Legend.Add("Heat Index", line)

	// Sunrise/Sunset vertical markers: draw tiny two-point lines.
	addVLine := func(hour float64, name string) error {
		if math.IsInf(plt.Y.Min, 0) || math.IsInf(plt.Y.Max, 0) {
			// If Y bounds are not set (all NaN), skip.
			return nil
		}
		vlpts := plotter.XYs{
			{X: hour, Y: plt.Y.Min},
			{X: hour, Y: plt.Y.Max},
		}
		vl, err := plotter.NewLine(vlpts)
		if err != nil {
			return err
		}
		// dashed marker
		vl.LineStyle.Dashes = []vg.Length{vg.Points(3), vg.Points(3)}
		plt.Add(vl)
		plt.Legend.Add(name, vl)
		return nil
	}
	_ = addVLine(float64(sunriseMin/60), "Sunrise")
	_ = addVLine(float64(sunsetMin/60), "Sunset")

	// Title suffix with windows (since Plot doesn't have Subtitle)
	plt.Title.Text += fmt.Sprintf("\nWalk: %s (~%dh), %s (~%dh) Avoid: %s-%s (%s)",
		win.MorningStart, win.MorningDurH, win.EveningStart, win.EveningDurH, win.AvoidStart, win.AvoidEnd, tz.String())

	// Save as PDF (6x4 inches)
	if err := plt.Save(6*vg.Inch, 4*vg.Inch, filename); err != nil {
		return err
	}
	return nil
}
