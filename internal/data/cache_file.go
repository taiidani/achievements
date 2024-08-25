package data

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/taiidani/achievements/internal/steam"
)

type FileCache struct {
}

func NewFileCache() *FileCache {
	// Ensure the cache directory exists
	_ = os.MkdirAll("_cache", 0777)
	return &FileCache{}
}

func (c *FileCache) GetGlobalAchievementPercentagesForApp(ctx context.Context, client *steam.Client, appID uint64) (*steam.GlobalAchievementPercentages, error) {
	// Check the cache to see if we've already scraped
	filename := filepath.Join("_cache", fmt.Sprintf("global_%d.json", appID))
	f, err := os.Open(filename)
	if err == nil {
		ret := &steam.GlobalAchievementPercentages{}
		err = json.NewDecoder(f).Decode(ret)
		return ret, err
	}

	// Nope! Build the cache
	ret, err := client.ISteamUserStats.GetGlobalAchievementPercentagesForApp(ctx, appID)
	if err != nil {
		return nil, err
	}

	f, err = os.Create(filename)
	if err != nil {
		return nil, err
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	err = enc.Encode(ret)

	return ret, err
}

func (c *FileCache) GetSchemaForGame(ctx context.Context, client *steam.Client, appID uint64) (*steam.GameSchema, error) {
	// Check the cache to see if we've already scraped
	filename := filepath.Join("_cache", fmt.Sprintf("schema_%d.json", appID))
	f, err := os.Open(filename)
	if err == nil {
		ret := &steam.GameSchema{}
		err = json.NewDecoder(f).Decode(ret)
		return ret, err
	}

	// Nope! Build the cache
	ret, err := client.ISteamUserStats.GetSchemaForGame(ctx, appID)
	if err != nil {
		return nil, err
	}

	f, err = os.Create(filename)
	if err != nil {
		return nil, err
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	err = enc.Encode(ret)

	return ret, err
}

func (c *FileCache) GetPlayerSummaries(ctx context.Context, client *steam.Client, userID string) (*steam.PlayerSummaries, error) {
	// Check the cache to see if we've already scraped
	filename := filepath.Join("_cache", fmt.Sprintf("player_%s_summary.json", userID))
	f, err := os.Open(filename)
	if err == nil {
		ret := &steam.PlayerSummaries{}
		err = json.NewDecoder(f).Decode(ret)
		return ret, err
	}

	// Nope! Build the cache
	ret, err := client.ISteamUser.GetPlayerSummaries(ctx, userID)
	if err != nil {
		return nil, err
	}

	f, err = os.Create(filename)
	if err != nil {
		return nil, err
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	err = enc.Encode(ret)

	return ret, err
}

func (c *FileCache) GetPlayerAchievements(ctx context.Context, client *steam.Client, userID string, appID uint64) (*steam.PlayerAchievements, error) {
	// Check the cache to see if we've already scraped
	filename := filepath.Join("_cache", fmt.Sprintf("player_%s_game_%d.json", userID, appID))
	f, err := os.Open(filename)
	if err == nil {
		ret := &steam.PlayerAchievements{}
		err = json.NewDecoder(f).Decode(ret)
		return ret, err
	}

	// Nope! Build the cache
	ret, err := client.ISteamUserStats.GetPlayerAchievements(ctx, userID, appID)
	if err != nil {
		// This will issue a Bad Request if no achievements exist for it
		// For now, emit an empty result so that we can cache the zero value
		ret = &steam.PlayerAchievements{
			PlayerStats: steam.PlayerStats{},
		}
	}

	f, err = os.Create(filename)
	if err != nil {
		return nil, err
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	err = enc.Encode(ret)

	return ret, err
}

func (c *FileCache) GetPlayerOwnedGames(ctx context.Context, client *steam.Client, userID string) (*steam.OwnedGames, error) {
	// Check the cache to see if we've already scraped
	filename := filepath.Join("_cache", fmt.Sprintf("player_%s_games.json", userID))
	f, err := os.Open(filename)
	if err == nil {
		ret := &steam.OwnedGames{}
		err = json.NewDecoder(f).Decode(ret)
		return ret, err
	}

	// Nope! Build the cache
	ret, err := client.IPlayerService.GetOwnedGames(ctx, userID)
	if err != nil {
		return nil, err
	}

	f, err = os.Create(filename)
	if err != nil {
		return nil, err
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	err = enc.Encode(ret)

	return ret, err
}

// func (c *FileCache) AppList(ctx context.Context, client *steam.Client) (*steam.AppList, error) {
// 	// Check the cache to see if we've already scraped
// 	filename := filepath.Join("_cache", fmt.Sprintf("apps.json"))
// 	f, err := os.Open(filename)
// 	if err == nil {
// 		ret := &steam.AppList{}
// 		err = json.NewDecoder(f).Decode(ret)
// 		return ret, err
// 	}

// 	apps, err := client.ISteamApps.GetAppList(ctx)
// 	if err != nil {
// 		return nil, err
// 	}

// 	f, err = os.Create(filename)
// 	if err != nil {
// 		return nil, err
// 	}

// 	enc := json.NewEncoder(f)
// 	enc.SetIndent("", "  ")
// 	err = enc.Encode(apps)

// 	return apps, err
// }
