package server

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/taiidani/achievements/internal/data"
)

type indexBag struct {
	baseBag
	User  data.User
	Games []struct {
		data.Game
		SteamID string
	}
}

func (s *Server) indexHandler(resp http.ResponseWriter, req *http.Request) {
	bag := indexBag{baseBag: newBag(req, "home")}

	// If no user has been set, display the welcome page
	if bag.SteamID == "" {
		renderHtml(resp, http.StatusOK, "index.gohtml", bag)
		return
	}

	// A user has been set. Gather their information!
	// My steamID is 76561197970932835
	user, err := s.backend.GetUser(req.Context(), bag.SteamID)
	if err != nil {
		// Attempt to resolve the user's vanity URL into an ID
		bag.SteamID, err = s.backend.ResolveVanityURL(req.Context(), bag.SteamID)
		if err != nil {
			errorResponse(resp, http.StatusNotFound, fmt.Errorf("could not resolve user id %q to a Steam User ID or Vanity URL: %w", bag.SteamID, err))
			return
		}

		user, err = s.backend.GetUser(req.Context(), bag.SteamID)
		if err != nil {
			errorResponse(resp, http.StatusNotFound, fmt.Errorf("could not get user data for %q: %w", bag.SteamID, err))
			return
		}
	}
	bag.User = user

	games, err := s.backend.GetGames(req.Context(), bag.SteamID)
	if err != nil {
		errorResponse(resp, http.StatusNotFound, err)
		return
	}

	for _, game := range games {
		bag.Games = append(bag.Games, struct {
			data.Game
			SteamID string
		}{
			Game:    game,
			SteamID: bag.SteamID,
		})
	}

	sort.Slice(bag.Games, func(i, j int) bool {
		return bag.Games[i].DisplayName < bag.Games[j].DisplayName
	})

	template := "games.gohtml"
	if req.Header.Get("HX-Request") != "" {
		template = "games-body.gohtml"
	}

	renderHtml(resp, http.StatusOK, template, bag)
}
