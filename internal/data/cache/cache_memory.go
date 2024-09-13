package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

type Memory struct {
	data map[string][]byte
	mx   sync.Mutex
}

var _ Cache = &Memory{}

func NewMemory() *Memory {
	return &Memory{
		data: map[string][]byte{},
		mx:   sync.Mutex{},
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
	c.mx.Lock()
	defer c.mx.Unlock()

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
				c.mx.Lock()
				defer c.mx.Unlock()
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

// Keys will return the set of keys in the cache matching the given "*" based pattern.
// WARNING: This implementation only supports a single wildcard.
func (c *Memory) Keys(ctx context.Context, pattern string) ([]string, error) {
	patterns := strings.Split(pattern, "*")
	if len(patterns) == 1 {
		// Special case: no wildcards exist
		if _, found := c.data[pattern]; found {
			return []string{pattern}, nil
		}
	} else if len(patterns) > 2 {
		return []string{}, fmt.Errorf("the memory cache does not support multiple wildcards")
	}

	ret := []string{}
	for key := range c.data {
		if !strings.HasPrefix(key, patterns[0]) {
			continue
		}
		if !strings.HasSuffix(key, patterns[1]) {
			continue
		}

		ret = append(ret, key)
	}

	return ret, nil
}
