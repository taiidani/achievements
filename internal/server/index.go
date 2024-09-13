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
	SteamID   string
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

	// If no user is logged in
	if bag.SessionUser == nil {
		renderHtml(resp, http.StatusOK, "index.gohtml", bag)
		return
	}

	bag.User = *bag.SessionUser
	bag.SteamID = bag.User.SteamID

	var err error
	bag.Games, bag.HasPinned, err = s.loadGamesList(req.Context(), bag.User.SteamID, bag.baseBag)
	if err != nil {
		errorResponse(resp, http.StatusNotFound, err)
		return
	}

	template := "games.gohtml"
	renderHtml(resp, http.StatusOK, template, bag)
}

func (s *Server) gamesHandler(resp http.ResponseWriter, req *http.Request) {
	bag := indexBag{baseBag: s.newBag(req, "home")}

	steamID := req.PathValue("steamid")
	if len(steamID) == 0 {
		errorResponse(resp, http.StatusBadRequest, fmt.Errorf("user ID is required"))
		return
	}

	// A user has been set. Gather their information!
	user, err := s.backend.GetUser(req.Context(), steamID)
	if err != nil {
		errorResponse(resp, http.StatusNotFound, fmt.Errorf("could not get user data for %q: %w", steamID, err))
		return
	}
	bag.User = user
	bag.SteamID = bag.User.SteamID

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
