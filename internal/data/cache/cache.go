package cache

import (
	"context"
	"time"
)

type Cache interface {
	Get(context.Context, string, any) error
	Set(context.Context, string, any, time.Duration) error
	Has(context.Context, string) (bool, error)
}
