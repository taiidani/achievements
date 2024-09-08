// package data documents the mechanisms for retrieving data from Steam as well as caching it locally.
//
// Many of the API calls are based upon endpoints documented at https://steamapi.xpaw.me/
package data

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/taiidani/achievements/internal/data/cache"
	"github.com/taiidani/achievements/internal/steam"
)

type Data struct {
	cache cache.Cache
	steam SteamHelper
}

type Game struct {
	ID              uint64
	DisplayName     string
	Icon            string
	PlaytimeForever time.Duration
	LastPlayed      time.Time
	LastPlayedSince time.Duration
}

type Achievements struct {
	Achievements                  []Achievement
	AchievementTotalCount         int
	AchievementUnlockedCount      int
	AchievementUnlockedPercentage int
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

func NewData(client *steam.Client, cache cache.Cache) *Data {
	return &Data{
		cache: cache,
		steam: *NewSteamHelper(client, cache),
	}
}

func (d *Data) GetUser(ctx context.Context, userID string) (User, error) {
	log := slog.With("user", userID)

	log.Debug("Retrieving user")
	playerSummaries, err := d.steam.GetPlayerSummaries(ctx, userID)
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
	log := slog.With("steam-id", userID)

	log.Debug("Retrieving user owned games")
	steamGames, err := d.steam.GetPlayerOwnedGames(ctx, userID)
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

		if err := d.populateGamePlaytime(&newData, &game); err != nil {
			log.Warn("Unable to populate playtime, leaving empty", "error", err)
		}

		ret = append(ret, newData)
	}

	return ret, nil
}

func (d *Data) GetGame(ctx context.Context, userID string, appID uint64) (Game, error) {
	log := slog.With("steam-id", userID, "app-id", appID)

	log.Debug("Retrieving user owned games")
	steamGames, err := d.steam.GetPlayerOwnedGames(ctx, userID)
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
		ID:          steamGame.AppID,
		Icon:        steamGame.ImgIconURL,
		DisplayName: steamGame.Name,
	}

	if err := d.populateGamePlaytime(&newData, &steamGame); err != nil {
		log.Warn("Unable to populate playtime, leaving empty", "error", err)
	}

	return newData, nil
}

func (d *Data) populateGamePlaytime(game *Game, steamGame *steam.OwnedGame) error {
	log := slog.With("game-name", steamGame.Name)
	game.PlaytimeForever = time.Duration(steamGame.PlaytimeForever) * time.Minute

	if steamGame.PlaytimeForever == 0 {
		log.Debug("Skipping unplayed game")
		return nil
	}

	// This number sometimes comes from the API when the game has not been played
	// Check for this alongside the zero value when determining if the game has been
	// played.
	const zeroTimePlayed = 86400

	if steamGame.RTimeLastPlayed > zeroTimePlayed {
		game.LastPlayed = time.Unix(int64(steamGame.RTimeLastPlayed), 0)
		game.LastPlayedSince = time.Since(game.LastPlayed)
	}

	return nil
}

func (d *Data) GetAchievements(ctx context.Context, userID string, gameID uint64) (Achievements, error) {
	log := slog.With("game-id", gameID)
	log.Debug("Retrieving schema for game")
	schema, err := d.steam.GetSchemaForGame(ctx, gameID)
	if err != nil {
		return Achievements{}, fmt.Errorf("unable to retrieve game schema: %w", err)
	} else if len(schema.Game.AvailableGameStats.Achievements) == 0 {
		log.Debug("Game has no achievements. Skipping.")
		return Achievements{}, nil
	}

	log.Debug("Retrieving global achievement percentages")
	globalAchievements, err := d.steam.GetGlobalAchievementPercentagesForApp(ctx, gameID)
	if err != nil {
		log.Warn("Unable to get achievements for game. Assuming no achievements and skipping.", "err", err)
		return Achievements{}, nil
	}

	log.Debug("Retrieving player achievements for game")
	playerAchievements, err := d.steam.GetPlayerAchievements(ctx, userID, gameID)
	if err != nil {
		log.Warn("Unable to get player achievements for game. Leaving empty.", "err", err)
		playerAchievements = &steam.PlayerAchievements{
			PlayerStats: steam.PlayerStats{
				Achievements: []steam.PlayerAchievement{},
			},
		}
	}

	ret := Achievements{}
	ret.AchievementTotalCount = len(playerAchievements.PlayerStats.Achievements)
	ret.AchievementUnlockedCount = 0

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
			ret.AchievementUnlockedCount++
		}

		bagAchievement.Icon = gameAchievement.Icon
		if !bagAchievement.Achieved {
			bagAchievement.Icon = gameAchievement.IconGray
		}

		ret.Achievements = append(ret.Achievements, bagAchievement)
	}

	ret.AchievementUnlockedPercentage = int((float64(ret.AchievementUnlockedCount) / float64(ret.AchievementTotalCount)) * 100)
	return ret, nil
}

func (d *Data) ResolveVanityURL(ctx context.Context, vanityURL string) (string, error) {
	slog.Debug("Resolving vanity URL", "name", vanityURL)
	vanity, err := d.steam.ResolveVanityURL(ctx, vanityURL)
	if err != nil {
		return "", fmt.Errorf("could not resolve vanity URL for player %q games: %w", vanityURL, err)
	}

	return vanity.Response.SteamID, nil
}
