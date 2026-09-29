package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileCacheGet(t *testing.T) {
	written := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		store  bool
		ttl    time.Duration
		age    time.Duration
		wantOK bool
	}{
		{"missing entry", false, time.Hour, 0, false},
		{"fresh entry", true, 12 * time.Hour, 11 * time.Hour, true},
		{"expired entry", true, 12 * time.Hour, 13 * time.Hour, false},
		{"zero ttl never expires", true, 0, 10000 * time.Hour, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			now := written.Add(tt.age)
			c := New(dir, func() time.Time { return now })
			if tt.store {
				if err := c.Put("r/k.json", []byte("x")); err != nil {
					t.Fatal(err)
				}
				if err := os.Chtimes(filepath.Join(dir, "r/k.json"), written, written); err != nil {
					t.Fatal(err)
				}
			}
			data, ok, err := c.Get("r/k.json", tt.ttl)
			if err != nil {
				t.Fatal(err)
			}
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && string(data) != "x" {
				t.Errorf("data = %q", data)
			}
		})
	}
}
