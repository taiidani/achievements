package cache

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	redis "github.com/redis/go-redis/v9"
)

type Redis struct {
	client *redis.Client
}

var _ Cache = &Redis{}

// NewRedis instantiates a new client
func NewRedis(addr string) *Redis {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	client := redis.NewClient(&redis.Options{Addr: addr})
	status := client.Ping(ctx)
	if status.Err() != nil {
		slog.Error("Failed to create Redis client", "addr", addr)
		panic(status.Err())
	}

	return &Redis{client: client}
}

// NewRedisSecureCache instantiates a new secure TLS client
func NewRedisSecureCache(host, port, user, password string, db int) *Redis {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	client := redis.NewClient(&redis.Options{
		Addr:      fmt.Sprintf("%s:%s", host, port),
		Username:  user,
		Password:  password,
		TLSConfig: &tls.Config{},
		DB:        db,
	})
	status := client.Ping(ctx)
	if status.Err() != nil {
		slog.Warn("Failed to create secure Redis client", "addr", client.Options().Addr)
		panic(status.Err())
	}

	return &Redis{client: client}
}

func (c *Redis) Get(ctx context.Context, key string, val any) error {
	resp := c.client.Get(ctx, key)
	if resp.Err() != nil {
		return resp.Err()
	}

	return json.Unmarshal([]byte(resp.Val()), val)
}

func (c *Redis) Set(ctx context.Context, key string, val any, ttl time.Duration) error {
	req, err := json.Marshal(val)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, req, ttl).Err()
}

func (c *Redis) Has(ctx context.Context, key string) (bool, error) {
	resp := c.client.Exists(ctx, key)
	if resp.Err() != nil {
		return false, resp.Err()
	}

	return resp.Val() > 0, nil
}
