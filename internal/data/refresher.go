package data

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/taiidani/achievements/internal/data/cache"
	"github.com/taiidani/achievements/internal/steam"
)

func Refresher(ctx context.Context, client *steam.Client, cache cache.Cache) {
	tick := time.NewTicker(time.Hour * 24)

	err := refreshData(ctx, client, cache)
	if err != nil {
		slog.Error("refresh cycle errored", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			slog.Info("Data refresher exited")
			return
		case <-tick.C:
			err = refreshData(ctx, client, cache)
			if err != nil {
				slog.Error("refresh cycle errored", "error", err)
			}
		}
	}
}

func refreshData(ctx context.Context, client *steam.Client, cache cache.Cache) error {
	slog.Info("Refreshing data")

	start := time.Now()
	defer func() {
		slog.Info("Refresh complete", "duration", time.Since(start))
	}()

	d := NewData(client, cache)

	appIDs, err := d.steam.GetSchemasInCache(ctx)
	if err != nil {
		return fmt.Errorf("unable to get schemas in cache: %w", err)
	}

	for _, appID := range appIDs {
		// We don't need the data itself; we're just refreshing its cache
		_, err := client.ISteamUserStats.GetSchemaForGame(ctx, appID)
		if err != nil {
			slog.Warn("Failed to get schema for game", "appID", appID, "error", err)
		}
	}

	return nil
}
