package data

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/taiidani/achievements/internal/data/cache"
	"github.com/taiidani/achievements/internal/steam"
)

type SteamHelper struct {
	client *steam.Client
	cache  cache.Cache
}

func NewSteamHelper(client *steam.Client, cache cache.Cache) *SteamHelper {
	return &SteamHelper{
		client: client,
		cache:  cache,
	}
}

func (c *SteamHelper) GetGlobalAchievementPercentagesForApp(ctx context.Context, appID uint64) (*steam.GlobalAchievementPercentages, error) {
	// Check the cache to see if we've already scraped
	key := fmt.Sprintf("game:%d:global", appID)
	ret := &steam.GlobalAchievementPercentages{}
	if err := c.cache.Get(ctx, key, ret); err == nil {
		return ret, nil
	}

	// Nope! Build the cache
	ret, err := c.client.ISteamUserStats.GetGlobalAchievementPercentagesForApp(ctx, appID)
	if err != nil {
		return nil, err
	}

	return ret, c.cache.Set(ctx, key, ret, time.Hour*24)
}

func (c *SteamHelper) GetSchemasInCache(ctx context.Context) ([]uint64, error) {
	gameSchemas, err := c.cache.Keys(ctx, "game:*:schema")
	if err != nil {
		return []uint64{}, fmt.Errorf("unable to scan cache for game schemas: %w", err)
	}

	r := regexp.MustCompile(`^game:(\d+):schema$`)
	ret := []uint64{}
	for _, key := range gameSchemas {
		match := r.FindStringSubmatch(key)
		if match == nil || len(match) < 2 {
			return ret, fmt.Errorf("unable to match returned key against regex")
		}

		matchInt, err := strconv.ParseUint(match[1], 10, 64)
		if err != nil {
			return ret, fmt.Errorf("returned match %q was not a valid integer: %w", match, err)
		}

		ret = append(ret, matchInt)
	}

	return ret, nil
}

func (c *SteamHelper) GetSchemaForGame(ctx context.Context, appID uint64) (*steam.GameSchema, error) {
	// Check the cache to see if we've already scraped
	key := fmt.Sprintf("game:%d:schema", appID)
	ret := &steam.GameSchema{}
	if err := c.cache.Get(ctx, key, ret); err == nil {
		return ret, nil
	}

	// Nope! Build the cache
	ret, err := c.client.ISteamUserStats.GetSchemaForGame(ctx, appID)
	if err != nil {
		return nil, err
	}

	return ret, c.cache.Set(ctx, key, ret, time.Hour*24*7)
}

func (c *SteamHelper) GetPlayerSummaries(ctx context.Context, userID string) (*steam.PlayerSummaries, error) {
	// Check the cache to see if we've already scraped
	key := fmt.Sprintf("player:%s:summary", userID)
	ret := &steam.PlayerSummaries{}
	if err := c.cache.Get(ctx, key, ret); err == nil {
		return ret, nil
	}

	// Nope! Build the cache
	ret, err := c.client.ISteamUser.GetPlayerSummaries(ctx, userID)
	if err != nil {
		return nil, err
	}

	return ret, c.cache.Set(ctx, key, ret, time.Hour)
}

func (c *SteamHelper) GetPlayerAchievements(ctx context.Context, userID string, appID uint64) (*steam.PlayerAchievements, error) {
	// Check the cache to see if we've already scraped
	key := fmt.Sprintf("player:%s:game:%d:achievements", userID, appID)
	ret := &steam.PlayerAchievements{}
	if err := c.cache.Get(ctx, key, ret); err == nil {
		return ret, nil
	}

	// Nope! Build the cache
	ret, err := c.client.ISteamUserStats.GetPlayerAchievements(ctx, userID, appID)
	if err != nil {
		// This will issue a Bad Request if no achievements exist for it
		// For now, emit an empty result so that we can cache the zero value
		ret = &steam.PlayerAchievements{
			PlayerStats: steam.PlayerStats{},
		}
	}

	return ret, c.cache.Set(ctx, key, ret, time.Hour)
}

func (c *SteamHelper) GetPlayerOwnedGames(ctx context.Context, userID string) (*steam.OwnedGames, error) {
	// Check the cache to see if we've already scraped
	key := fmt.Sprintf("player:%s:games", userID)
	ret := &steam.OwnedGames{}
	if err := c.cache.Get(ctx, key, ret); err == nil {
		return ret, nil
	}

	// Nope! Build the cache
	ret, err := c.client.IPlayerService.GetOwnedGames(ctx, userID)
	if err != nil {
		return nil, err
	}

	return ret, c.cache.Set(ctx, key, ret, time.Hour*24)
}

func (c *SteamHelper) ResolveVanityURL(ctx context.Context, vanityURL string) (*steam.VanityURLResponse, error) {
	// Check the cache to see if we've already scraped
	key := fmt.Sprintf("player:%s:vanity", vanityURL)
	ret := &steam.VanityURLResponse{}
	if err := c.cache.Get(ctx, key, ret); err == nil {
		return ret, nil
	}

	// Nope! Build the cache
	ret, err := c.client.ISteamUser.ResolveVanityURL(ctx, vanityURL)
	if err != nil {
		return nil, err
	}

	return ret, c.cache.Set(ctx, key, ret, time.Hour*24)
}
