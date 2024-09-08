package steam

import (
	"bufio"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const (
	// openIDProvider defines the OpenID provider URL when authenticating a Steam user.
	//
	// See https://steamcommunity.com/dev for more information.
	openIDProvider = "https://steamcommunity.com/openid"

	// openIDClaimPrefix defines the prefix preceding the SteamID of the logged in user.
	// The Claimed ID format is: https://steamcommunity.com/openid/id/<steamid>.
	//
	// See https://steamcommunity.com/dev for more information.
	openIDClaimPrefix = "https://steamcommunity.com/openid/id/"
)

type OpenIDClient struct {
	apiKey string
}

func NewOpenIDClient() *OpenIDClient {
	return &OpenIDClient{
		apiKey: os.Getenv("STEAM_KEY"),
	}
}

func (c *OpenIDClient) LoginURL(realm string, loginPage string) (*url.URL, error) {

	query := url.Values{}
	query.Set("openid.ns", "http://specs.openid.net/auth/2.0")
	query.Set("openid.mode", "checkid_setup")
	query.Set("openid.return_to", loginPage)
	query.Set("openid.realm", realm)
	query.Set("openid.identity", "http://specs.openid.net/auth/2.0/identifier_select")
	query.Set("openid.claimed_id", "http://specs.openid.net/auth/2.0/identifier_select")

	ret, err := url.Parse(openIDProvider + "/login?" + query.Encode())
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (c *OpenIDClient) GetSteamID(params url.Values) (string, error) {
	const claimedIDKey = "openid.claimed_id"
	const signatureKey = "openid.sig"

	claimedID := params.Get(claimedIDKey)
	if claimedID == "" {
		return "", fmt.Errorf("required %q parameter not present", claimedIDKey)
	}

	sig := params.Get(signatureKey)
	if sig == "" {
		return "", fmt.Errorf("required %q parameter not present", signatureKey)
	}

	userID := strings.TrimPrefix(claimedID, openIDClaimPrefix)
	return userID, nil
}

func (c *OpenIDClient) Validate(params url.Values) error {
	query := url.Values{}
	query.Set("openid.sig", params.Get("openid_sig"))
	query.Set("openid.ns", "http://specs.openid.net/auth/2.0")

	signed := strings.Split(params.Get("openid_signed"), ",")

	// Get all the params that were sent back as part of the signature
	for _, item := range signed {
		key := "openid." + item
		query.Set(key, params.Get(key))
	}

	// Ensure that Steam understands that we are performing a validation
	query.Set("openid.mode", "check_authentication")

	// Send the validation request to Steam
	req, err := http.NewRequest(http.MethodPost, openIDProvider+"/login?"+query.Encode(), nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Expected response:
	// ns:http://specs.openid.net/auth/2.0
	// is_valid:false
	body := bufio.NewScanner(resp.Body)
	for body.Scan() {
		if body.Text() == "is_valid:false" {
			return nil
		}
	}

	return fmt.Errorf("unable to validate signature for OpenID response")
}
