package server

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/taiidani/achievements/internal/models"
)

const (
	steamIDHeaderKey = "steam-id"
)

func (s *Server) sessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("%s %s\n", r.Method, r.URL.Path)

		// Sanitize well-known header values
		if !DevMode {
			r.Header.Del(steamIDHeaderKey)
		}

		// Is the user ID in the session?
		sess := models.Session{}
		err := s.session.Get(r, &sess)
		if err != nil {
			slog.Warn("Unable to retrieve session", "error", err)
		} else {
			r.Header.Add(steamIDHeaderKey, sess.SteamID)
		}

		next.ServeHTTP(w, r)
	})
}
