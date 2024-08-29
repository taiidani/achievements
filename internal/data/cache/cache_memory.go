package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type Memory struct {
	data map[string][]byte
}

var _ Cache = &Memory{}

func NewMemory() *Memory {
	return &Memory{
		data: map[string][]byte{},
	}
}

func (c *Memory) Get(_ context.Context, key string, val any) error {
	d, ok := c.data[key]
	if !ok {
		return fmt.Errorf("key not found")
	}

	return json.Unmarshal(d, val)
}

// Set will create a cache file for the given key.
// TODO: Support TTL based expirations
func (c *Memory) Set(ctx context.Context, key string, val any, ttl time.Duration) error {
	var err error
	c.data[key], err = json.Marshal(val)

	if err == nil {
		// Enforce the TTL by deleting the key after the specified duration
		// Note that this is an unsafe operation. If the key is set again within the timeframe
		// it will NOT extend the duration.
		go func() {
			select {
			case <-ctx.Done():
				return
			case <-time.After(ttl):
				delete(c.data, key)
			}
		}()
	}

	return err
}

func (c *Memory) Has(ctx context.Context, key string) (bool, error) {
	_, ok := c.data[key]
	return ok, nil
}
