package openmeteo

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

const forecastFixture = `{
  "timezone": "Asia/Nicosia",
  "hourly": {
    "time": ["2025-07-01T06:00", "2025-07-01T07:00", "2025-07-01T08:00"],
    "temperature_2m": [24.5, null, 30],
    "relative_humidity_2m": [60, 55, 50]
  },
  "daily": {
    "sunrise": ["2025-07-01T05:40", "2025-07-02T05:41", "2025-07-03T05:44"],
    "sunset": ["2025-07-01T20:00", "2025-07-02T20:01", "2025-07-03T20:02"]
  }
}`

const archiveFixture = `{
  "timezone": "Europe/Belgrade",
  "hourly": {
    "time": ["2025-01-01T00:00", "2025-06-15T13:00"],
    "temperature_2m": [-2, 31],
    "relative_humidity_2m": [90, 40]
  }
}`

type fakeDoer struct {
	status int
	body   string
	urls   []string
}

func (d *fakeDoer) Do(req *http.Request) (*http.Response, error) {
	d.urls = append(d.urls, req.URL.String())
	return &http.Response{
		StatusCode: d.status,
		Status:     http.StatusText(d.status),
		Body:       io.NopCloser(strings.NewReader(d.body)),
	}, nil
}

type memCache map[string][]byte

func (c memCache) Get(key string, _ time.Duration) ([]byte, bool, error) {
	data, ok := c[key]
	return data, ok, nil
}

func (c memCache) Put(key string, data []byte) error {
	c[key] = data
	return nil
}

var paphos = Point{Lat: 34.7768, Lon: 32.4245}

func TestSourceLastWeek(t *testing.T) {
	tests := []struct {
		name       string
		cached     memCache
		status     int
		wantErr    bool
		wantCalls  int
		wantTZ     string
		wantTemps  []float64
		wantSunMin [2]int
	}{
		{
			name:       "fetches, skips null hours, takes median sun times",
			cached:     memCache{},
			status:     http.StatusOK,
			wantCalls:  1,
			wantTZ:     "Asia/Nicosia",
			wantTemps:  []float64{24.5, 30},
			wantSunMin: [2]int{5*60 + 41, 20*60 + 1},
		},
		{
			name:       "cache hit skips http",
			cached:     memCache{"forecast/34.7768_32.4245.json": []byte(forecastFixture)},
			status:     http.StatusInternalServerError,
			wantCalls:  0,
			wantTZ:     "Asia/Nicosia",
			wantTemps:  []float64{24.5, 30},
			wantSunMin: [2]int{5*60 + 41, 20*60 + 1},
		},
		{
			name:      "http error is returned",
			cached:    memCache{},
			status:    http.StatusBadGateway,
			wantErr:   true,
			wantCalls: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doer := &fakeDoer{status: tt.status, body: forecastFixture}
			got, err := NewSource(NewClient(doer), tt.cached, time.Hour, 0).LastWeek(paphos)
			if len(doer.urls) != tt.wantCalls {
				t.Fatalf("http calls = %d, want %d", len(doer.urls), tt.wantCalls)
			}
			if tt.wantErr {
				if err == nil {
					t.Fatal("want error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.Timezone != tt.wantTZ {
				t.Errorf("timezone = %q, want %q", got.Timezone, tt.wantTZ)
			}
			if len(got.Samples) != len(tt.wantTemps) {
				t.Fatalf("samples = %d, want %d", len(got.Samples), len(tt.wantTemps))
			}
			for i, s := range got.Samples {
				if s.TempC != tt.wantTemps[i] {
					t.Errorf("sample %d temp = %v, want %v", i, s.TempC, tt.wantTemps[i])
				}
				if s.Time.Location().String() != tt.wantTZ {
					t.Errorf("sample %d in %s, want %s", i, s.Time.Location(), tt.wantTZ)
				}
			}
			if [2]int{got.SunriseMin, got.SunsetMin} != tt.wantSunMin {
				t.Errorf("sun = %d/%d, want %v", got.SunriseMin, got.SunsetMin, tt.wantSunMin)
			}
		})
	}
}

func TestSourceYear(t *testing.T) {
	tests := []struct {
		name        string
		year        int
		wantKey     string
		wantURLPart []string
		wantHours   []int
	}{
		{
			name:        "requests whole calendar year and caches by point",
			year:        2025,
			wantKey:     "archive/34.7768_32.4245-2025.json",
			wantURLPart: []string{"archive-api", "start_date=2025-01-01", "end_date=2025-12-31"},
			wantHours:   []int{0, 13},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doer := &fakeDoer{status: http.StatusOK, body: archiveFixture}
			cache := memCache{}
			got, err := NewSource(NewClient(doer), cache, time.Hour, 0).Year(paphos, tt.year)
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := cache[tt.wantKey]; !ok {
				t.Errorf("cache keys = %v, want %s", keys(cache), tt.wantKey)
			}
			for _, part := range tt.wantURLPart {
				if !strings.Contains(doer.urls[0], part) {
					t.Errorf("url %s lacks %s", doer.urls[0], part)
				}
			}
			if len(got) != len(tt.wantHours) {
				t.Fatalf("samples = %d, want %d", len(got), len(tt.wantHours))
			}
			for i, s := range got {
				if s.Time.Hour() != tt.wantHours[i] {
					t.Errorf("sample %d hour = %d, want %d", i, s.Time.Hour(), tt.wantHours[i])
				}
			}
		})
	}
}

func keys(c memCache) []string {
	out := make([]string, 0, len(c))
	for k := range c {
		out = append(out, k)
	}
	return out
}
