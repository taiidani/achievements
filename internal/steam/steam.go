package steam

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

type Client struct {
	IPlayerService  *iPlayerService
	ISteamApps      *iSteamAppsService
	ISteamUser      *iSteamUserService
	ISteamUserStats *iSteamUserStatsService
}

type service struct {
	client *http.Client
}

const (
	apiVersion01   = "v1"
	apiVersion02   = "v2"
	steamAPIHost   = "api.steampowered.com"
	steamAPIScheme = "https"
)

func NewClient() *Client {
	svc := &service{
		client: http.DefaultClient,
	}

	return &Client{
		IPlayerService:  newIPlayerService(svc),
		ISteamApps:      newISteamAppsService(svc),
		ISteamUser:      newISteamUserService(svc),
		ISteamUserStats: newISteamUserStatsService(svc),
	}
}

func (s *service) url(api string, method string, version string, values url.Values) url.URL {
	ret := url.URL{
		Scheme: steamAPIScheme,
		Host:   steamAPIHost,
	}

	ret.Path, _ = url.JoinPath(api, method, version)
	values.Add("key", os.Getenv("STEAM_KEY"))
	ret.RawQuery = values.Encode()
	return ret
}

func (s *service) httpError(resp *http.Response, err error) error {
	if err != nil {
		return fmt.Errorf("could not submit request: %w", err)
	}

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	// Non-OK response. Let's grab the body to troubleshoot
	_, _ = io.Copy(os.Stderr, resp.Body)

	var ret error
	switch resp.StatusCode {
	case http.StatusNotFound:
		ret = fmt.Errorf("endpoint not found")
	case http.StatusMethodNotAllowed:
		ret = fmt.Errorf("method not allowed")
	case http.StatusBadGateway:
		ret = fmt.Errorf("bad gateway")
	case http.StatusBadRequest:
		ret = fmt.Errorf("bad request")
	case http.StatusForbidden:
		ret = fmt.Errorf("forbidden")
	default:
		ret = fmt.Errorf("unknown API response %d", resp.StatusCode)
	}

	return ret
}
