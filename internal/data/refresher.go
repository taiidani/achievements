package data

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/taiidani/achievements/internal/data/cache"
	"github.com/taiidani/achievements/internal/steam"
)

// refresherData contains a list of UserIDs to regularly refresh.
var refresherData = []string{
	// taiidani
	"76561197970932835",
}

func Refresher(ctx context.Context, client *steam.Client, cache cache.Cache) {
	tick := time.NewTicker(time.Hour * 24)

	for _, userID := range refresherData {
		if err := refreshData(ctx, client, cache, userID); err != nil {
			slog.Error("Failed to refresh data", "error", err)
		}
	}

	for {
		select {
		case <-ctx.Done():
			slog.Info("Data refresher exited")
			return
		case <-tick.C:
			for _, userID := range refresherData {
				if err := refreshData(ctx, client, cache, userID); err != nil {
					slog.Error("Failed to refresh data", "error", err)
				}
			}
		}
	}
}

func refreshData(ctx context.Context, client *steam.Client, cache cache.Cache, userID string) error {
	log := slog.With("user", userID)
	d := NewData(client, cache)

	log.Info("Refreshing data")
	start := time.Now()
	defer func() {
		log.Info("Refresh complete", "duration", time.Since(start))
	}()

	log.Debug("Retrieving user owned games")
	steamGames, err := d.cache.GetPlayerOwnedGames(ctx, userID)
	if err != nil {
		return fmt.Errorf("could not query for player %q games: %w", userID, err)
	}

	for _, steamGame := range steamGames.Response.Games {
		_, err := d.GetGame(ctx, userID, steamGame.AppID)
		if err != nil {
			return fmt.Errorf("could not refresh data for game %q: %w", steamGame.AppID, err)
		}
	}

	return nil
}
