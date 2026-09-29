package weather

import (
	"fmt"
	"sort"
	"time"

	"github.com/nsr888/heat24/internal/heat"
)

const timeLayout = "2006-01-02T15:04"

type response struct {
	Timezone string `json:"timezone"`
	Hourly   struct {
		Time               []string   `json:"time"`
		Temperature2m      []*float64 `json:"temperature_2m"`
		RelativeHumidity2m []*float64 `json:"relative_humidity_2m"`
	} `json:"hourly"`
	Daily struct {
		Sunrise []string `json:"sunrise"`
		Sunset  []string `json:"sunset"`
	} `json:"daily"`
}

func (r *response) samples(loc *time.Location) ([]heat.Sample, error) {
	h := r.Hourly
	if len(h.Temperature2m) != len(h.Time) || len(h.RelativeHumidity2m) != len(h.Time) {
		return nil, fmt.Errorf("hourly arrays differ in length")
	}
	out := make([]heat.Sample, 0, len(h.Time))
	for i, ts := range h.Time {
		if h.Temperature2m[i] == nil || h.RelativeHumidity2m[i] == nil {
			continue
		}
		t, err := time.ParseInLocation(timeLayout, ts, loc)
		if err != nil {
			return nil, err
		}
		out = append(out, heat.Sample{Time: t, TempC: *h.Temperature2m[i], RH: *h.RelativeHumidity2m[i]})
	}
	return out, nil
}

func (r *response) medianSunTimes(loc *time.Location) (sunriseMin, sunsetMin int, err error) {
	sunrise, err := minutesOfDay(r.Daily.Sunrise, loc)
	if err != nil {
		return 0, 0, err
	}
	sunset, err := minutesOfDay(r.Daily.Sunset, loc)
	if err != nil {
		return 0, 0, err
	}
	if len(sunrise) == 0 || len(sunset) == 0 {
		return 0, 0, fmt.Errorf("no sunrise/sunset data")
	}
	return median(sunrise), median(sunset), nil
}

func minutesOfDay(values []string, loc *time.Location) ([]int, error) {
	out := make([]int, 0, len(values))
	for _, v := range values {
		t, err := time.ParseInLocation(timeLayout, v, loc)
		if err != nil {
			return nil, err
		}
		out = append(out, t.Hour()*60+t.Minute())
	}
	return out, nil
}

func median(values []int) int {
	sorted := append([]int(nil), values...)
	sort.Ints(sorted)
	n := len(sorted)
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}
