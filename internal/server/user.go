package server

import (
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/taiidani/achievements/internal/data"
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
	if bag.SteamID != "" {
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
	if bag.SteamID != "" {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	// Confirm that the request was completed and signed by Steam
	params := r.URL.Query()
	if !params.Has("openid.sig") {
		http.Error(w, "Request must be signed", http.StatusBadRequest)
		return
	}

	// Validate that this did indeed come from Steam
	oauth := steam.NewOpenIDClient()
	err := oauth.Validate(params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// It's valid! Get the SteamID from the parameters
	steamID, err := oauth.GetSteamID(params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Now build the session and set the cookie
	sess := data.Session{
		SteamID: steamID,
	}
	sessionKey := s.buildSessionKey()
	err = s.backend.SetSession(r.Context(), sessionKey, sess)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cookie := http.Cookie{
		Name:     "session",
		Value:    sessionKey,
		Secure:   !DevMode,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   int(data.DefaultSessionExpiration.Seconds()),
	}
	http.SetCookie(w, &cookie)
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
