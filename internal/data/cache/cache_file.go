package cache

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type File struct {
}

var _ Cache = &File{}

func NewFile() *File {
	// Ensure the cache directory exists
	_ = os.MkdirAll("_cache", 0777)
	return &File{}
}

func (c *File) Get(ctx context.Context, key string, val any) error {
	filename := c.normalizeKey(key)
	f, err := os.Open(filename)
	if err != nil {
		return err
	}

	return json.NewDecoder(f).Decode(val)
}

// Set will create a cache file for the given key.
// TODO: Support TTL based expirations
func (c *File) Set(ctx context.Context, key string, val any, _ time.Duration) error {
	filename := c.normalizeKey(key)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(val)
}

func (c *File) Has(ctx context.Context, key string) (bool, error) {
	filename := c.normalizeKey(key)
	_, err := os.Open(filename)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (c *File) normalizeKey(key string) string {
	return filepath.Join("_cache", strings.ReplaceAll(key, ":", "_"))
}
