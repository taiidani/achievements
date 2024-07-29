package data

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/taiidani/achievements/internal/steam"
)

type CachedData struct {
	Games []Game
}

type Game struct {
	ID                            uint64
	DisplayName                   string
	Icon                          string
	Achievements                  []Achievement
	AchievementTotalCount         int
	AchievementUnlockedCount      int
	AchievementUnlockedPercentage int
	PlaytimeForever               time.Duration
	LastPlayed                    time.Time
	LastPlayedSince               time.Duration
}

type Achievement struct {
	Name             string
	Description      string
	Hidden           bool
	Icon             string
	GlobalPercentage float64
	Achieved         bool
	UnlockedOn       *time.Time
}

type User struct {
	SteamID     string
	Name        string
	ProfileURL  string
	AvatarURL   string
	LastLogoff  time.Time
	TimeCreated time.Time
}

var Data = map[string]*CachedData{}

func init() {
	// Ensure the cache directory exists
	_ = os.MkdirAll("_cache", 0777)
}

func RefreshData(ctx context.Context, userID string) error {
	log := slog.With("user", userID)

	log.Info("Refreshing data")
	start := time.Now()
	defer func() {
		log.Info("Refresh complete", "duration", time.Since(start))
	}()

	newData := &CachedData{}

	client := steam.NewClient()

	log.Debug("Retrieving user owned games")
	steamGames, err := cachedGetPlayerOwnedGames(ctx, client, userID)
	if err != nil {
		return fmt.Errorf("could not query for player %q games: %w", userID, err)
	}

	for _, steamGame := range steamGames.Response.Games {
		game, err := GetGame(ctx, userID, steamGame.AppID)
		if err != nil {
			return fmt.Errorf("could not refresh data for game %q: %w", steamGame.AppID, err)
		}

		newData.Games = append(newData.Games, game)
	}

	Data[userID] = newData
	return nil
}

func GetUser(ctx context.Context, userID string) (User, error) {
	log := slog.With("user", userID)
	client := steam.NewClient()

	log.Debug("Retrieving user")
	playerSummaries, err := cachedGetPlayerSummaries(ctx, client, userID)
	if err != nil {
		return User{}, fmt.Errorf("could not query for player %q games: %w", userID, err)
	}

	if len(playerSummaries.Response.Players) == 0 {
		return User{}, fmt.Errorf("no user found for ID %q", userID)
	}
	user := playerSummaries.Response.Players[0]

	newData := User{
		SteamID:     user.SteamID,
		Name:        user.PersonaName,
		ProfileURL:  user.ProfileURL,
		AvatarURL:   user.AvatarFull,
		TimeCreated: time.Unix(int64(user.TimeCreated), 0),
	}
	if user.LastLogoff > 0 {
		newData.LastLogoff = time.Unix(int64(user.LastLogoff), 0)
	}

	return newData, nil
}

func GetGames(ctx context.Context, userID string) ([]Game, error) {
	log := slog.With("user-id", userID)
	client := steam.NewClient()

	log.Debug("Retrieving user owned games")
	steamGames, err := cachedGetPlayerOwnedGames(ctx, client, userID)
	if err != nil {
		return nil, fmt.Errorf("could not query for player %q games: %w", userID, err)
	}

	ret := []Game{}
	for _, game := range steamGames.Response.Games {
		log := log.With("game-id", game.AppID, "title", game.Name)
		if game.PlaytimeForever == 0 {
			continue
		}

		newData := Game{
			ID:              game.AppID,
			DisplayName:     game.Name,
			Achievements:    []Achievement{},
			PlaytimeForever: time.Duration(game.PlaytimeForever) * time.Minute,
		}
		if game.RTimeLastPlayed > 0 {
			newData.LastPlayed = time.Unix(int64(game.RTimeLastPlayed), 0)
			newData.LastPlayedSince = time.Since(newData.LastPlayed)
		}

		playerAchievements, err := cachedGetPlayerAchievements(ctx, client, userID, game.AppID)
		if err != nil {
			log.Warn("Unable to get player achievements for game. Skipping game.", "err", err)
			continue
		} else if len(playerAchievements.PlayerStats.Achievements) == 0 {
			log.Debug("Game has no achievements. Skipping game")
			continue
		}

		newData.AchievementTotalCount = len(playerAchievements.PlayerStats.Achievements)
		newData.AchievementUnlockedCount = 0
		for _, achievement := range playerAchievements.PlayerStats.Achievements {
			if achievement.Achieved > 0 {
				newData.AchievementUnlockedCount++
			}
		}

		newData.AchievementUnlockedPercentage = int((float64(newData.AchievementUnlockedCount) / float64(newData.AchievementTotalCount)) * 100)

		ret = append(ret, newData)
	}

	return ret, nil
}

func GetGame(ctx context.Context, userID string, appID uint64) (Game, error) {
	log := slog.With("user", userID, "app-id", appID)
	client := steam.NewClient()

	log.Debug("Retrieving user owned games")
	steamGames, err := cachedGetPlayerOwnedGames(ctx, client, userID)
	if err != nil {
		return Game{}, fmt.Errorf("could not query for player %q games: %w", userID, err)
	}

	var steamGame steam.OwnedGame
	for _, game := range steamGames.Response.Games {
		if game.AppID == appID {
			steamGame = game
			break
		}
	}

	log = log.With("game-name", steamGame.Name)
	newData := Game{
		ID:           steamGame.AppID,
		DisplayName:  steamGame.Name,
		Achievements: []Achievement{},
	}

	if steamGame.PlaytimeForever == 0 {
		log.Debug("Skipping unplayed game")
		return newData, nil
	}

	log.Debug("Retrieving schema for game")
	schema, err := cachedGetSchemaForGame(ctx, client, steamGame.AppID)
	if err != nil {
		return newData, fmt.Errorf("unable to retrieve game schema: %w", err)
	} else if len(schema.Game.AvailableGameStats.Achievements) == 0 {
		log.Debug("Game has no achievements. Skipping.")
		return newData, nil
	}

	log.Debug("Retrieving global achievement percentages")
	globalAchievements, err := cachedGetGlobalAchievementPercentagesForApp(ctx, client, steamGame.AppID)
	if err != nil {
		log.Warn("Unable to get achievements for game. Assuming no achievements and skipping.", "err", err)
		return newData, nil
	}

	log.Debug("Retrieving player achievements for game")
	playerAchievements, err := cachedGetPlayerAchievements(ctx, client, userID, steamGame.AppID)
	if err != nil {
		log.Warn("Unable to get player achievements for game. Leaving empty.", "err", err)
		playerAchievements = &steam.PlayerAchievements{
			PlayerStats: steam.PlayerStats{
				Achievements: []steam.PlayerAchievement{},
			},
		}
	}
	newData.AchievementTotalCount = len(playerAchievements.PlayerStats.Achievements)
	newData.AchievementUnlockedCount = 0

	for _, gameAchievement := range schema.Game.AvailableGameStats.Achievements {
		bagAchievement := Achievement{
			Name:        gameAchievement.DisplayName,
			Description: gameAchievement.Description,
			Hidden:      gameAchievement.Hidden > 0,
			Achieved:    false,
			UnlockedOn:  nil,
		}

		// Get the global completion percentage
		for _, globalAchievement := range globalAchievements.AchievementPercentages.Achievements {
			if gameAchievement.Name == globalAchievement.Name {
				bagAchievement.GlobalPercentage = globalAchievement.Percent
			}
		}

		// And this player's unlock, if present
		for _, playerAchievement := range playerAchievements.PlayerStats.Achievements {
			if gameAchievement.Name == playerAchievement.APIName {
				bagAchievement.Achieved = playerAchievement.Achieved > 0
				bagAchievement.Hidden = !bagAchievement.Achieved && bagAchievement.Hidden

				if playerAchievement.UnlockTime > 0 {
					tm := time.Unix(int64(playerAchievement.UnlockTime), 0)
					bagAchievement.UnlockedOn = &tm
				}
			}
		}
		if bagAchievement.Achieved {
			newData.AchievementUnlockedCount++
		}

		bagAchievement.Icon = gameAchievement.Icon
		if !bagAchievement.Achieved {
			bagAchievement.Icon = gameAchievement.IconGray
		}

		newData.Achievements = append(newData.Achievements, bagAchievement)
	}

	newData.AchievementUnlockedPercentage = int((float64(newData.AchievementUnlockedCount) / float64(newData.AchievementTotalCount)) * 100)

	return newData, nil
}

func Refresher(ctx context.Context) {
	tick := time.NewTicker(time.Hour * 24)

	for userID := range Data {
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
			for userID := range Data {
				if err := RefreshData(ctx, userID); err != nil {
					slog.Error("Failed to refresh data", "error", err)
				}
			}
		}
	}
}

func cachedGetGlobalAchievementPercentagesForApp(ctx context.Context, client *steam.Client, appID uint64) (*steam.GlobalAchievementPercentages, error) {
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

func cachedGetSchemaForGame(ctx context.Context, client *steam.Client, appID uint64) (*steam.GameSchema, error) {
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

func cachedGetPlayerSummaries(ctx context.Context, client *steam.Client, userID string) (*steam.PlayerSummaries, error) {
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

func cachedGetPlayerAchievements(ctx context.Context, client *steam.Client, userID string, appID uint64) (*steam.PlayerAchievements, error) {
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

func cachedGetPlayerOwnedGames(ctx context.Context, client *steam.Client, userID string) (*steam.OwnedGames, error) {
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

// func cachedAppList(ctx context.Context, client *steam.Client) (*steam.AppList, error) {
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
