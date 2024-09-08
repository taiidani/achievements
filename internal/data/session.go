package data

import (
	"context"
	"time"
)

type Session struct {
	SteamID string `json:"SteamID"`
}

const DefaultSessionExpiration = time.Hour * 24 * 90

func (d *Data) GetSession(ctx context.Context, key string) (*Session, error) {
	key = "session:" + key
	if ok, _ := d.cache.Has(ctx, key); !ok {
		return nil, nil
	}

	ret := &Session{}
	err := d.cache.Get(ctx, key, ret)
	return ret, err
}

func (d *Data) SetSession(ctx context.Context, key string, sess Session) error {
	key = "session:" + key
	return d.cache.Set(ctx, key, sess, DefaultSessionExpiration)
}
