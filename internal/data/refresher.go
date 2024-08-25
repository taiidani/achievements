package data

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/taiidani/achievements/internal/steam"
)

type CachedData struct {
	Games []Game
}

var data = map[string]*CachedData{}

func Refresher(ctx context.Context) {
	tick := time.NewTicker(time.Hour * 24)

	for userID := range data {
		if err := RefreshData(ctx, userID); err != nil {
			slog.Error("Failed to refresh data", "error", err)
		}
	}

	for {
		select {
		case <-ctx.Done():
			slog.Info("Data refresher exited")
			return
		case <-tick.C:
			for userID := range data {
				if err := RefreshData(ctx, userID); err != nil {
					slog.Error("Failed to refresh data", "error", err)
				}
			}
		}
	}
}

func RefreshData(ctx context.Context, userID string) error {
	log := slog.With("user", userID)

	cache := NewFileCache()
	d := NewData(cache)

	log.Info("Refreshing data")
	start := time.Now()
	defer func() {
		log.Info("Refresh complete", "duration", time.Since(start))
	}()

	newData := &CachedData{}

	client := steam.NewClient()

	log.Debug("Retrieving user owned games")
	steamGames, err := cache.GetPlayerOwnedGames(ctx, client, userID)
	if err != nil {
		return fmt.Errorf("could not query for player %q games: %w", userID, err)
	}

	for _, steamGame := range steamGames.Response.Games {
		game, err := d.GetGame(ctx, userID, steamGame.AppID)
		if err != nil {
			return fmt.Errorf("could not refresh data for game %q: %w", steamGame.AppID, err)
		}

		newData.Games = append(newData.Games, game)
	}

	data[userID] = newData
	return nil
}
