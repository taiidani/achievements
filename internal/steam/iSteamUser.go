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

type iSteamUserService struct {
	*service
}

func newISteamUserService(service *service) *iSteamUserService {
	return &iSteamUserService{
		service: service,
	}
}

type PlayerSummaries struct {
	Response struct {
		Players []Player `json:"players"`
	} `json:"response"`
}

type Player struct {
	SteamID                  string `json:"steamid"`
	CommunityVisibilityState uint64 `json:"communityvisibilitystate"`
	ProfileState             uint64 `json:"profilestate"`
	PersonaName              string `json:"personaname"`
	ProfileURL               string `json:"profileurl"`
	Avatar                   string `json:"avatar"`
	AvatarMedium             string `json:"avatarmedium"`
	AvatarFull               string `json:"avatarfull"`
	AvatarHash               string `json:"avatarhash"`
	LastLogoff               uint64 `json:"lastlogoff"`
	PersonaState             uint64 `json:"personastate"`
	RealName                 string `json:"realname"`
	PrimaryClanID            string `json:"primaryclanid"`
	TimeCreated              uint64 `json:"timecreated"`
	PersonaStateFlags        uint64 `json:"personastateflags"`
	LocCountryCode           string `json:"loccountrycode"`
	LocStateCode             string `json:"locstatecode"`
	LocCityID                uint64 `json:"loccityid"`
}

func (c *iSteamUserService) GetPlayerSummaries(ctx context.Context, userID ...string) (*PlayerSummaries, error) {
	query := url.Values{}
	query.Add("steamids", strings.Join(userID, ","))
	target := c.url("ISteamUser", "GetPlayerSummaries", apiVersion02, query)
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
	// slog.DebugContext(ctx, "Response received", "body", respBody.String())

	ret := &PlayerSummaries{}
	err = json.Unmarshal([]byte(respBody.String()), ret)
	if err != nil {
		return nil, fmt.Errorf("unable to parse response: %w", err)
	}

	return ret, err
}

type VanityURLResponse struct {
	Response struct {
		SteamID string `json:"steamid"`
		Success uint64 `json:"success"`
	} `json:"response"`
}

func (c *iSteamUserService) ResolveVanityURL(ctx context.Context, vanityURL string) (*VanityURLResponse, error) {
	query := url.Values{}
	query.Add("vanityurl", vanityURL)
	target := c.url("ISteamUser", "ResolveVanityURL", apiVersion01, query)
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
	// slog.DebugContext(ctx, "Response received", "body", respBody.String())

	ret := &VanityURLResponse{}
	err = json.Unmarshal([]byte(respBody.String()), ret)
	if err != nil {
		return nil, fmt.Errorf("unable to parse response: %w", err)
	}

	return ret, err
}
