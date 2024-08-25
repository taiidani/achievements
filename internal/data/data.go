package data

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/taiidani/achievements/internal/steam"
)

type Data struct {
	cache DataCache
}

type DataCache interface {
	GetGlobalAchievementPercentagesForApp(ctx context.Context, client *steam.Client, appID uint64) (*steam.GlobalAchievementPercentages, error)
	GetSchemaForGame(ctx context.Context, client *steam.Client, appID uint64) (*steam.GameSchema, error)
	GetPlayerSummaries(ctx context.Context, client *steam.Client, userID string) (*steam.PlayerSummaries, error)
	GetPlayerAchievements(ctx context.Context, client *steam.Client, userID string, appID uint64) (*steam.PlayerAchievements, error)
	GetPlayerOwnedGames(ctx context.Context, client *steam.Client, userID string) (*steam.OwnedGames, error)
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

func NewData(cache DataCache) *Data {
	return &Data{
		cache: cache,
	}
}

func (d *Data) GetUser(ctx context.Context, userID string) (User, error) {
	log := slog.With("user", userID)
	client := steam.NewClient()

	log.Debug("Retrieving user")
	playerSummaries, err := d.cache.GetPlayerSummaries(ctx, client, userID)
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

func (d *Data) GetGames(ctx context.Context, userID string) ([]Game, error) {
	log := slog.With("user-id", userID)
	client := steam.NewClient()

	log.Debug("Retrieving user owned games")
	steamGames, err := d.cache.GetPlayerOwnedGames(ctx, client, userID)
	if err != nil {
		return nil, fmt.Errorf("could not query for player %q games: %w", userID, err)
	}

	ret := []Game{}
	for _, game := range steamGames.Response.Games {
		if game.PlaytimeForever == 0 {
			continue
		}

		newData := Game{
			ID:          game.AppID,
			DisplayName: game.Name,
			Icon:        game.ImgIconURL,
		}

		ret = append(ret, newData)
	}

	return ret, nil
}

func (d *Data) GetGame(ctx context.Context, userID string, appID uint64) (Game, error) {
	log := slog.With("user", userID, "app-id", appID)
	client := steam.NewClient()

	log.Debug("Retrieving user owned games")
	steamGames, err := d.cache.GetPlayerOwnedGames(ctx, client, userID)
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
		ID:              steamGame.AppID,
		Icon:            steamGame.ImgIconURL,
		DisplayName:     steamGame.Name,
		Achievements:    []Achievement{},
		PlaytimeForever: time.Duration(steamGame.PlaytimeForever) * time.Minute,
	}

	if steamGame.PlaytimeForever == 0 {
		log.Debug("Skipping unplayed game")
		return newData, nil
	}

	if steamGame.RTimeLastPlayed > 0 {
		newData.LastPlayed = time.Unix(int64(steamGame.RTimeLastPlayed), 0)
		newData.LastPlayedSince = time.Since(newData.LastPlayed)
	}

	log.Debug("Retrieving schema for game")
	schema, err := d.cache.GetSchemaForGame(ctx, client, steamGame.AppID)
	if err != nil {
		return newData, fmt.Errorf("unable to retrieve game schema: %w", err)
	} else if len(schema.Game.AvailableGameStats.Achievements) == 0 {
		log.Debug("Game has no achievements. Skipping.")
		return newData, nil
	}

	log.Debug("Retrieving global achievement percentages")
	globalAchievements, err := d.cache.GetGlobalAchievementPercentagesForApp(ctx, client, steamGame.AppID)
	if err != nil {
		log.Warn("Unable to get achievements for game. Assuming no achievements and skipping.", "err", err)
		return newData, nil
	}

	log.Debug("Retrieving player achievements for game")
	playerAchievements, err := d.cache.GetPlayerAchievements(ctx, client, userID, steamGame.AppID)
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
