package internal

import (
	"math"
	"sort"
	"time"
)

type HourStats struct {
	TempSum float64
	RHSum   float64
	Count   int
}

type Averages struct {
	ByHourTemp []float64 // 24 values
	ByHourRH   []float64 // 24 values
}

func PerHourMeans(om *openMeteoResp) (Averages, *time.Location, error) {
	loc, err := time.LoadLocation(om.Timezone)
	if err != nil {
		return Averages{}, nil, err
	}
	stats := make([]HourStats, 24)

	n := len(om.Hourly.Time)
	for i := range n {
		t, err := time.ParseInLocation("2006-01-02T15:04", om.Hourly.Time[i], loc)
		if err != nil {
			return Averages{}, nil, err
		}
		h := t.Hour()
		stats[h].TempSum += om.Hourly.Temperature2m[i]
		stats[h].RHSum += om.Hourly.RelativeHumidity2m[i]
		stats[h].Count++
	}

	avg := Averages{
		ByHourTemp: make([]float64, 24),
		ByHourRH:   make([]float64, 24),
	}
	for h := range 24 {
		if stats[h].Count > 0 {
			avg.ByHourTemp[h] = stats[h].TempSum / float64(stats[h].Count)
			avg.ByHourRH[h] = stats[h].RHSum / float64(stats[h].Count)
		} else {
			avg.ByHourTemp[h] = math.NaN()
			avg.ByHourRH[h] = math.NaN()
		}
	}
	return avg, loc, nil
}

// Heat Index in °C from temp °C and RH % (NOAA regression via °F).
func heatIndexC(tempC, rh float64) float64 {
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

func HourHeatIndex(avg Averages) []float64 {
	hi := make([]float64, 24)
	for h := range 24 {
		hi[h] = heatIndexC(avg.ByHourTemp[h], avg.ByHourRH[h])
	}
	return hi
}

func MedianSunTimes(om *openMeteoResp, loc *time.Location) (sunriseMin, sunsetMin int, err error) {
	var sunr, suns []int
	for i := range om.Daily.Sunrise {
		rs, err := time.ParseInLocation("2006-01-02T15:04", om.Daily.Sunrise[i], loc)
		if err != nil {
			return 0, 0, err
		}
		ss, err := time.ParseInLocation("2006-01-02T15:04", om.Daily.Sunset[i], loc)
		if err != nil {
			return 0, 0, err
		}
		sunr = append(sunr, rs.Hour()*60+rs.Minute())
		suns = append(suns, ss.Hour()*60+ss.Minute())
	}
	sort.Ints(sunr)
	sort.Ints(suns)
	mid := len(sunr) / 2
	return sunr[mid], suns[mid], nil
}
