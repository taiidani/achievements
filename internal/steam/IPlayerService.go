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

type iPlayerService struct {
	*service
}

func newIPlayerService(service *service) *iPlayerService {
	return &iPlayerService{
		service: service,
	}
}

type OwnedGames struct {
	Response OwnedGamesResponse `json:"response"`
}

type OwnedGamesResponse struct {
	GameCount uint64      `json:"game_count"`
	Games     []OwnedGame `json:"games"`
}

type OwnedGame struct {
	AppID                  uint64   `json:"appid"`
	Name                   string   `json:"name"`
	ImgIconURL             string   `json:"img_icon_url"`
	PlaytimeForever        uint64   `json:"playtime_forever"`
	PlaytimeWindowsForever uint64   `json:"playtime_windows_forever"`
	PlaytimeMacForever     uint64   `json:"playtime_mac_forever"`
	PlaytimeLinuxForever   uint64   `json:"playtime_linux_forever"`
	PlaytimeDeckForever    uint64   `json:"playtime_deck_forever"`
	RTimeLastPlayed        uint64   `json:"rtime_last_played"`
	ContentDescriptorIDs   []uint64 `json:"content_descriptorids"`
	PlaytimeDisconnected   uint64   `json:"playtime_disconnected"`

	// HasCommunityVisibleStats bool     `json:"has_community_visible_stats"`
	// HasWorkshop              bool     `json:"has_workshop"`
	// HasMarket                bool     `json:"has_market"`
	// HasDLC                   bool     `json:"has_dlc"`
}

func (c *iPlayerService) GetOwnedGames(ctx context.Context, userID string) (*OwnedGames, error) {
	query := url.Values{}
	query.Add("steamid", userID)
	query.Add("include_appinfo", "true")
	// query.Add("include_extended_appinfo", "true")
	query.Add("include_played_free_games", "false")
	query.Add("include_free_sub", "false")
	target := c.url("IPlayerService", "GetOwnedGames", apiVersion01, query)
	slog.DebugContext(ctx, "URL formed", "url", target.String())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("could not format request: %w", err)
	}

	resp, err := c.client.Do(req)
	if resp != nil && resp.Close {
		defer resp.Body.Close()
	}
	if err := c.httpError(resp, err); err != nil {
		return nil, fmt.Errorf("request error: %w", err)
	}

	respBody := &strings.Builder{}
	_, _ = io.Copy(respBody, resp.Body)
	// fmt.Println(respBody.String())

	ret := &OwnedGames{}
	err = json.Unmarshal([]byte(respBody.String()), ret)
	if err != nil {
		return nil, fmt.Errorf("unable to parse response: %w", err)
	}

	return ret, err
}
