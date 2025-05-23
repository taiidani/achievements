package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/taiidani/achievements/internal/models"
	"github.com/taiidani/achievements/internal/steam"
)

type userChangeBag struct {
	baseBag
}

func (s *Server) userChangeHandler(w http.ResponseWriter, r *http.Request) {
	bag := userChangeBag{baseBag: s.newBag(r, "change-user")}

	template := "change-user.gohtml"
	renderHtml(w, http.StatusOK, template, bag)
}

type userLoginBag struct {
	baseBag
	SteamLoginURL *url.URL
}

func (s *Server) userLoginHandler(w http.ResponseWriter, r *http.Request) {
	bag := userLoginBag{baseBag: s.newBag(r, "user-login")}

	// If the user is already logged in, redirect them to the homepage
	if bag.SessionUser != nil {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	oauth := steam.NewOpenIDClient()
	var err error
	bag.SteamLoginURL, err = oauth.LoginURL(s.publicURL, s.publicURL+"/user/login/steam")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	template := "login.gohtml"
	renderHtml(w, http.StatusOK, template, bag)
}

type userLoginSteamBag struct {
	baseBag
}

func (s *Server) userLoginSteamHandler(w http.ResponseWriter, r *http.Request) {
	bag := userLoginSteamBag{baseBag: s.newBag(r, "user-login-steam")}

	// If the user is already logged in, redirect them to the homepage
	if bag.SessionUser != nil {
		slog.Warn("User already signed in; redirecting")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	// Confirm that the request was completed and signed by Steam
	params := r.URL.Query()
	if !params.Has("openid.sig") {
		slog.Warn("Sign in failed due to missing OpenID signature")
		http.Error(w, "Request must be signed", http.StatusBadRequest)
		return
	}

	// Validate that this did indeed come from Steam
	oauth := steam.NewOpenIDClient()
	err := oauth.Validate(params)
	if err != nil {
		slog.Warn("Sign in failed due to invalid OAuth signature")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// It's valid! Get the SteamID from the parameters
	steamID, err := oauth.GetSteamID(params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	slog.Info("Signed in with Steam", "steamid", steamID)

	// Now build the session and set the cookie
	sess := models.Session{
		SteamID: steamID,
	}

	cookie, err := s.session.Create(r.Context(), sess)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, cookie)
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func (s *Server) userLogoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:    "session",
		Value:   "",
		Path:    "/",
		Expires: time.Unix(0, 0),
	})

	slog.Info("User logged out")
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func (s *Server) userLookupHandler(w http.ResponseWriter, r *http.Request) {
	steamID := r.FormValue("steam-id")
	if len(steamID) == 0 {
		errorResponse(w, http.StatusBadRequest, fmt.Errorf("invalid Steam ID provided"))
		return
	}

	// Lookup the user, confirming their Steam ID
	_, err := s.backend.GetUser(r.Context(), steamID)
	if err != nil {
		// Attempt to resolve the user's vanity URL into an ID
		steamID, err = s.backend.ResolveVanityURL(r.Context(), steamID)
		if err != nil {
			errorResponse(w, http.StatusNotFound, fmt.Errorf("could not resolve user id %q to a Steam User ID or Vanity URL: %w", steamID, err))
			return
		}

		_, err = s.backend.GetUser(r.Context(), steamID)
		if err != nil {
			errorResponse(w, http.StatusNotFound, fmt.Errorf("could not get user data for %q: %w", steamID, err))
			return
		}
	}

	http.Redirect(w, r, fmt.Sprintf("/user/%s/games", steamID), http.StatusTemporaryRedirect)
}
