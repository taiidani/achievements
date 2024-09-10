package server

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
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
		cookie, err := r.Cookie("session")
		if err == nil {
			sess, err := s.backend.GetSession(r.Context(), cookie.Value)
			if err != nil {
				slog.Warn("Unable to retrieve session", "key", cookie.Value, "error", err)
			} else if sess != nil {
				r.Header.Add(steamIDHeaderKey, sess.SteamID)
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) buildSessionKey() string {
	key := uuid.New()
	return key.String()
}
