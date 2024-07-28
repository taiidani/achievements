package steam

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

type iSteamUserStatsService struct {
	*service
}

func newISteamUserStatsService(service *service) *iSteamUserStatsService {
	return &iSteamUserStatsService{
		service: service,
	}
}

type GlobalAchievementPercentages struct {
	AchievementPercentages AchievementPercentages `json:"achievementpercentages"`
}

type AchievementPercentages struct {
	Achievements []Achievement `json:"achievements"`
}

type Achievement struct {
	Name    string  `json:"name"`
	Percent float64 `json:"percent"`
}

func (c *iSteamUserStatsService) GetGlobalAchievementPercentagesForApp(ctx context.Context, appID uint64) (*GlobalAchievementPercentages, error) {
	query := url.Values{}
	query.Add("gameid", fmt.Sprintf("%d", appID))
	target := c.url("ISteamUserStats", "GetGlobalAchievementPercentagesForApp", apiVersion02, query)
	slog.DebugContext(ctx, "URL formed", "url", target.String())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("could not format request: %w", err)
	}

	resp, err := c.client.Do(req)
	if resp.Close {
		defer resp.Body.Close()
	}
	if err := c.httpError(resp, err); err != nil {
		return nil, fmt.Errorf("request error: %w", err)
	}

	respBody := &strings.Builder{}
	_, _ = io.Copy(respBody, resp.Body)
	// slog.DebugContext(ctx, "Response received", "body", respBody.String())

	ret := &GlobalAchievementPercentages{}
	err = json.Unmarshal([]byte(respBody.String()), ret)
	if err != nil {
		return nil, fmt.Errorf("unable to parse response: %w", err)
	}

	return ret, err
}

type GameSchema struct {
	Game Game `json:"game"`
}

type Game struct {
	Name               string    `json:"gameName"`
	Version            string    `json:"gameVersion"`
	AvailableGameStats GameStats `json:"availableGameStats"`
}

type GameStats struct {
	Achievements []GameAchievement `json:"achievements"`
}

type GameAchievement struct {
	Name         string `json:"name"`
	DefaultValue uint64 `json:"defaultvalue"`
	DisplayName  string `json:"displayName"`
	Hidden       uint64 `json:"hidden"`
	Description  string `json:"description"`
	Icon         string `json:"icon"`
	IconGray     string `json:"icongray"`
}

func (c *iSteamUserStatsService) GetSchemaForGame(ctx context.Context, appID uint64) (*GameSchema, error) {
	query := url.Values{}
	query.Add("appid", fmt.Sprintf("%d", appID))
	target := c.url("ISteamUserStats", "GetSchemaForGame", apiVersion02, query)
	slog.DebugContext(ctx, "URL formed", "url", target.String())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("could not format request: %w", err)
	}

	resp, err := c.client.Do(req)
	if resp.Close {
		defer resp.Body.Close()
	}
	if err := c.httpError(resp, err); err != nil {
		return nil, fmt.Errorf("request error: %w", err)
	}

	respBody := &strings.Builder{}
	_, _ = io.Copy(respBody, resp.Body)
	// slog.DebugContext(ctx, "Response received", "body", respBody.String())

	ret := &GameSchema{}
	err = json.Unmarshal([]byte(respBody.String()), ret)
	if err != nil {
		return nil, fmt.Errorf("unable to parse response: %w", err)
	}

	return ret, err
}

type PlayerAchievements struct {
	PlayerStats PlayerStats `json:"playerstats"`
	Success     bool        `json:"success"`
}

type PlayerStats struct {
	SteamID      string              `json:"steamid"`
	GameName     string              `json:"gamename"`
	Achievements []PlayerAchievement `json:"achievements"`
}

type PlayerAchievement struct {
	APIName    string `json:"apiname"`
	Achieved   uint64 `json:"achieved"`
	UnlockTime uint64 `json:"unlocktime"`
}

func (c *iSteamUserStatsService) GetPlayerAchievements(ctx context.Context, userID string, appID uint64) (*PlayerAchievements, error) {
	query := url.Values{}
	query.Add("steamid", userID)
	query.Add("appid", fmt.Sprintf("%d", appID))
	target := c.url("ISteamUserStats", "GetPlayerAchievements", apiVersion01, query)
	slog.DebugContext(ctx, "URL formed", "url", target.String())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("could not format request: %w", err)
	}

	resp, err := c.client.Do(req)
	if resp.Close {
		defer resp.Body.Close()
	}
	if err := c.httpError(resp, err); err != nil {
		return nil, fmt.Errorf("request error: %w", err)
	}

	respBody := &strings.Builder{}
	_, _ = io.Copy(respBody, resp.Body)
	// slog.DebugContext(ctx, "Response received", "body", respBody.String())

	ret := &PlayerAchievements{}
	err = json.Unmarshal([]byte(respBody.String()), ret)
	if err != nil {
		return nil, fmt.Errorf("unable to parse response: %w", err)
	}

	return ret, err
}
