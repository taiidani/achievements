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

type iSteamAppsService struct {
	*service
}

func newISteamAppsService(service *service) *iSteamAppsService {
	return &iSteamAppsService{
		service: service,
	}
}

type AppList struct {
	AppList Apps `json:"applist"`
}

type Apps struct {
	Apps []App `json:"apps"`
}

type App struct {
	AppID uint64 `json:"appid"`
	Name  string `json:"name"`
}

func (c *iSteamAppsService) GetAppList(ctx context.Context) (*AppList, error) {
	target := c.url("ISteamApps", "GetAppList", apiVersion02, url.Values{})
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
	// fmt.Println(respBody.String())

	ret := &AppList{}
	err = json.Unmarshal([]byte(respBody.String()), ret)
	if err != nil {
		return nil, fmt.Errorf("unable to parse response: %w", err)
	}

	return ret, err
}

type SDRConfig struct {
	Revision       uint64         `json:"revision"`
	Pops           map[string]any `json:"pops"`
	Certs          []string       `json:"certs"`
	P2PShareID     map[string]any `json:"p2p_share_ip"`
	RelayPublicKey string         `json:"relay_public_key"`
	RevokedKeys    []string       `json:"revoked_keys"`
	TypicalPings   []any          `json:"typical_pings"`
	Success        bool           `json:"success"`
}

func (c *iSteamAppsService) GetSDRConfig(ctx context.Context, appID uint64) (*SDRConfig, error) {
	query := url.Values{}
	query.Add("appid", fmt.Sprintf("%d", appID))
	target := c.url("ISteamApps", "GetSDRConfig", apiVersion01, query)

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
	// fmt.Println(respBody.String())

	ret := &SDRConfig{}
	err = json.Unmarshal([]byte(respBody.String()), ret)
	if err != nil {
		return nil, fmt.Errorf("unable to parse response: %w", err)
	}

	return ret, err
}
