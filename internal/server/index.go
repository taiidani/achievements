package server

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/taiidani/achievements/internal/data"
)

type indexBag struct {
	baseBag
	UserID string
	User   data.User
	Games  []struct {
		data.Game
		UserID string
	}
}

func (s *Server) indexHandler(resp http.ResponseWriter, req *http.Request) {
	bag := indexBag{}
	bag.Page = "home"
	bag.UserID = req.URL.Query().Get("user-id")

	if bag.UserID != "" {
		// My userID is 76561197970932835
		user, err := s.backend.GetUser(req.Context(), bag.UserID)
		if err != nil {
			// Attempt to resolve the user's vanity URL into an ID
			bag.UserID, err = s.backend.ResolveVanityURL(req.Context(), bag.UserID)
			if err != nil {
				errorResponse(resp, http.StatusNotFound, fmt.Errorf("could not resolve user id %q to a Steam User ID or Vanity URL: %w", bag.UserID, err))
				return
			}

			user, err = s.backend.GetUser(req.Context(), bag.UserID)
			if err != nil {
				errorResponse(resp, http.StatusNotFound, fmt.Errorf("could not get user data for %q: %w", bag.UserID, err))
				return
			}
		}
		bag.User = user

		games, err := s.backend.GetGames(req.Context(), bag.UserID)
		if err != nil {
			errorResponse(resp, http.StatusNotFound, err)
			return
		}

		for _, game := range games {
			bag.Games = append(bag.Games, struct {
				data.Game
				UserID string
			}{
				Game:   game,
				UserID: bag.UserID,
			})
		}
	}

	sort.Slice(bag.Games, func(i, j int) bool {
		return bag.Games[i].DisplayName < bag.Games[j].DisplayName
	})

	template := "index.gohtml"
	if req.Header.Get("HX-Request") != "" {
		template = "index-body.gohtml"
	}

	renderHtml(resp, http.StatusOK, template, bag)
}
