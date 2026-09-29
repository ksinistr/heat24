package heat

import (
	"math"
	"time"
)

const hybridThresholdC = 27.0

type Sample struct {
	Time  time.Time
	TempC float64
	RH    float64
}

// NOAA Rothfusz regression, computed in °F and returned in °C.
func IndexC(tempC, rh float64) float64 {
	t := tempC*9.0/5.0 + 32.0
	r := rh

	hi := -42.379 +
		2.04901523*t +
		10.14333127*r -
		0.22475541*t*r -
		6.83783e-3*t*t -
		5.481717e-2*r*r +
		1.22874e-3*t*t*r +
		8.5282e-4*t*r*r -
		1.99e-6*t*t*r*r

	if r < 13 && t >= 80 && t <= 112 {
		hi -= ((13 - r) / 4.0) * math.Sqrt((17.0-math.Abs(t-95.0))/17.0)
	}
	if r > 85 && t >= 80 && t <= 87 {
		hi += ((r - 85.0) / 10.0) * ((87.0 - t) / 5.0)
	}

	return (hi - 32.0) * 5.0 / 9.0
}

func HybridC(tempC, rh float64) float64 {
	if tempC < hybridThresholdC {
		return tempC
	}
	return IndexC(tempC, rh)
}

// Averages temperature and humidity per hour of day, then derives the Hybrid Index.
// Hours without samples are NaN.
func HourlyHybrid(samples []Sample) [24]float64 {
	var tempSum, rhSum [24]float64
	var count [24]int
	for _, s := range samples {
		h := s.Time.Hour()
		tempSum[h] += s.TempC
		rhSum[h] += s.RH
		count[h]++
	}
	var out [24]float64
	for h := range 24 {
		if count[h] == 0 {
			out[h] = math.NaN()
			continue
		}
		n := float64(count[h])
		out[h] = HybridC(tempSum[h]/n, rhSum[h]/n)
	}
	return out
}

// Averages the Hybrid Index of each sample by hour of day (rows) and month (columns).
// Cells without samples are NaN.
func HourMonthGrid(samples []Sample) [24][12]float64 {
	var sum [24][12]float64
	var count [24][12]int
	for _, s := range samples {
		h, m := s.Time.Hour(), int(s.Time.Month())-1
		sum[h][m] += HybridC(s.TempC, s.RH)
		count[h][m]++
	}
	var out [24][12]float64
	for h := range 24 {
		for m := range 12 {
			if count[h][m] == 0 {
				out[h][m] = math.NaN()
				continue
			}
			out[h][m] = sum[h][m] / float64(count[h][m])
		}
	}
	return out
}

func InMonth(samples []Sample, month time.Month) []Sample {
	var out []Sample
	for _, s := range samples {
		if s.Time.Month() == month {
			out = append(out, s)
		}
	}
	return out
}
