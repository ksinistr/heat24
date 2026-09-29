package cache

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type FileCache struct {
	dir string
	now func() time.Time
}

func New(dir string, now func() time.Time) *FileCache {
	return &FileCache{dir: dir, now: now}
}

// A zero ttl means the entry never expires.
func (c *FileCache) Get(key string, ttl time.Duration) ([]byte, bool, error) {
	path := filepath.Join(c.dir, key)
	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if ttl > 0 && c.now().Sub(info.ModTime()) > ttl {
		return nil, false, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false, err
	}
	return data, true, nil
}

func (c *FileCache) Put(key string, data []byte) error {
	path := filepath.Join(c.dir, key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
