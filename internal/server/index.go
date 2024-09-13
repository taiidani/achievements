package server

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"sort"

	"github.com/taiidani/achievements/internal/data"
)

type indexBag struct {
	baseBag
	User      data.User
	HasPinned bool
	Games     []indexBagGame
}

type indexBagGame struct {
	data.Game
	Pinned bool
}

func (s *Server) indexHandler(resp http.ResponseWriter, req *http.Request) {
	bag := indexBag{baseBag: s.newBag(req, "home")}

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

	bag.Games, bag.HasPinned, err = s.loadGamesList(req.Context(), user.SteamID, bag.baseBag)
	if err != nil {
		errorResponse(resp, http.StatusNotFound, err)
		return
	}

	template := "games.gohtml"
	renderHtml(resp, http.StatusOK, template, bag)
}

func (s *Server) loadGamesList(ctx context.Context, steamID string, bag baseBag) ([]indexBagGame, bool, error) {
	ret := []indexBagGame{}
	retPinned := false

	games, err := s.backend.GetGames(ctx, steamID)
	if err != nil {
		return ret, false, err
	}

	for _, game := range games {
		if ok, err := s.backend.HasAchievements(ctx, game.ID); err != nil {
			return ret, false, fmt.Errorf("could not retrieve achievements: %w", err)
		} else if !ok {
			continue
		}

		bagGame := indexBagGame{
			Game:   game,
			Pinned: bag.Session != nil && slices.Contains(bag.Session.Pinned, game.ID),
		}

		if bagGame.Pinned {
			retPinned = true
		}

		ret = append(ret, bagGame)
	}

	sort.Slice(ret, func(i, j int) bool {
		return ret[i].DisplayName < ret[j].DisplayName
	})

	return ret, retPinned, nil
}
