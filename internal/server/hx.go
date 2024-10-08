package server

import (
	"fmt"
	"net/http"
	"slices"
	"strconv"

	"github.com/taiidani/achievements/internal/data"
)

type hxGameRowBag struct {
	baseBag
	GameID       uint64
	Achievements data.Achievements
}

func (s *Server) hxGameRowHandler(w http.ResponseWriter, r *http.Request) {
	bag := hxGameRowBag{baseBag: s.newBag(r, "")}

	steamID := r.PathValue("steamid")
	if len(steamID) == 0 {
		errorResponse(w, http.StatusBadRequest, fmt.Errorf("user ID is required"))
		return
	}

	gameIDString := r.PathValue("gameid")
	bag.GameID, _ = strconv.ParseUint(gameIDString, 10, 64)
	if bag.GameID == 0 {
		errorResponse(w, http.StatusBadRequest, fmt.Errorf("invalid Game ID provided"))
		return
	}

	achievements, err := s.backend.GetAchievements(r.Context(), steamID, bag.GameID)
	if err != nil {
		errorResponse(w, http.StatusNotFound, err)
		return
	}
	bag.Achievements = achievements

	renderHtml(w, http.StatusOK, "hx-achievement-progress.gohtml", bag)
}

type hxGamePinBag struct {
	baseBag
	SteamID   string
	HasPinned bool
	Games     []indexBagGame
}

func (s *Server) hxGamePinHandler(w http.ResponseWriter, r *http.Request) {
	bag := hxGamePinBag{baseBag: s.newBag(r, "")}

	bag.SteamID = r.PathValue("steamid")
	if len(bag.SteamID) == 0 {
		errorResponse(w, http.StatusBadRequest, fmt.Errorf("user ID is required"))
		return
	}

	gameIDString := r.PathValue("gameid")
	gameID, _ := strconv.ParseUint(gameIDString, 10, 64)
	if gameID == 0 {
		errorResponse(w, http.StatusNotFound, fmt.Errorf("game-id required for pinning"))
		return
	}

	switch r.Method {
	case http.MethodPost:
		if !slices.Contains(bag.Session.Pinned, gameID) {
			bag.Session.Pinned = append(bag.Session.Pinned, gameID)
			_ = s.backend.SetSession(r.Context(), bag.SessionKey, *bag.Session)
		}
	case http.MethodDelete:
		if ix := slices.Index(bag.Session.Pinned, gameID); ix >= 0 {
			bag.Session.Pinned = slices.Delete(bag.Session.Pinned, ix, ix+1)
			_ = s.backend.SetSession(r.Context(), bag.SessionKey, *bag.Session)
		}
	}

	var err error
	bag.Games, bag.HasPinned, err = s.loadGamesList(r.Context(), bag.SteamID, bag.baseBag)
	if err != nil {
		errorResponse(w, http.StatusNotFound, err)
		return
	}

	renderHtml(w, http.StatusOK, "games-pinned.gohtml", bag)
}
