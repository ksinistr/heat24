package heat

import (
	"math"
	"testing"
	"time"
)

func TestHybridC(t *testing.T) {
	tests := []struct {
		name  string
		temp  float64
		rh    float64
		want  float64
		delta float64
	}{
		{"cool returns air temperature", 20, 90, 20, 0},
		{"just below threshold", 26.9, 80, 26.9, 0},
		{"hot and humid uses heat index", 32, 70, 40.41, 0.01},
		{"hot and dry uses heat index", 35, 10, 31.92, 0.01},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HybridC(tt.temp, tt.rh)
			if math.Abs(got-tt.want) > tt.delta {
				t.Errorf("HybridC(%v, %v) = %.2f, want %.2f±%.1f", tt.temp, tt.rh, got, tt.want, tt.delta)
			}
		})
	}
}

func TestHourlyHybrid(t *testing.T) {
	day := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		samples []Sample
		hour    int
		want    float64
	}{
		{
			name: "averages samples of the same hour",
			samples: []Sample{
				{Time: day.Add(6 * time.Hour), TempC: 18, RH: 50},
				{Time: day.Add(30 * time.Hour), TempC: 22, RH: 70},
			},
			hour: 6,
			want: 20,
		},
		{
			name:    "hour without samples is NaN",
			samples: []Sample{{Time: day, TempC: 18, RH: 50}},
			hour:    5,
			want:    math.NaN(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HourlyHybrid(tt.samples)[tt.hour]
			if !sameFloat(got, tt.want) {
				t.Errorf("hour %d = %v, want %v", tt.hour, got, tt.want)
			}
		})
	}
}

func TestHourMonthGrid(t *testing.T) {
	tests := []struct {
		name    string
		samples []Sample
		hour    int
		month   time.Month
		want    float64
	}{
		{
			name: "averages within hour and month",
			samples: []Sample{
				{Time: time.Date(2024, 1, 3, 8, 0, 0, 0, time.UTC), TempC: 4, RH: 80},
				{Time: time.Date(2024, 1, 9, 8, 0, 0, 0, time.UTC), TempC: 8, RH: 80},
				{Time: time.Date(2024, 2, 9, 8, 0, 0, 0, time.UTC), TempC: 30, RH: 80},
			},
			hour:  8,
			month: time.January,
			want:  6,
		},
		{
			name:    "empty cell is NaN",
			samples: nil,
			hour:    0,
			month:   time.March,
			want:    math.NaN(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HourMonthGrid(tt.samples)[tt.hour][tt.month-1]
			if !sameFloat(got, tt.want) {
				t.Errorf("cell = %v, want %v", got, tt.want)
			}
		})
	}
}

func sameFloat(a, b float64) bool {
	if math.IsNaN(a) || math.IsNaN(b) {
		return math.IsNaN(a) && math.IsNaN(b)
	}
	return math.Abs(a-b) < 1e-9
}

func TestInMonth(t *testing.T) {
	jan := Sample{Time: time.Date(2025, 1, 31, 23, 0, 0, 0, time.UTC)}
	feb := Sample{Time: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)}
	tests := []struct {
		name    string
		samples []Sample
		month   time.Month
		want    int
	}{
		{"keeps only matching month", []Sample{jan, feb, jan}, time.January, 2},
		{"no match is empty", []Sample{jan}, time.March, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := len(InMonth(tt.samples, tt.month)); got != tt.want {
				t.Errorf("len = %d, want %d", got, tt.want)
			}
		})
	}
}
