package server

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/taiidani/achievements/internal/data"
)

type gameBag struct {
	baseBag
	SteamID      string
	Game         data.Game
	Achievements data.Achievements
}

func (s *Server) gameHandler(resp http.ResponseWriter, req *http.Request) {
	bag := gameBag{baseBag: s.newBag(req, "game")}

	// taiidani's steamID is 76561197970932835
	bag.SteamID = req.PathValue("steamid")
	if len(bag.SteamID) == 0 {
		errorResponse(resp, http.StatusBadRequest, fmt.Errorf("invalid User ID provided"))
		return
	}

	gameIDString := req.PathValue("gameid")
	gameID, _ := strconv.ParseUint(gameIDString, 10, 64)
	if gameID == 0 {
		errorResponse(resp, http.StatusBadRequest, fmt.Errorf("invalid Game ID provided"))
		return
	}

	game, err := s.backend.GetGame(req.Context(), bag.SteamID, gameID)
	if err != nil {
		errorResponse(resp, http.StatusNotFound, err)
		return
	}
	bag.Game = game

	bag.Achievements, err = s.backend.GetAchievements(req.Context(), bag.SteamID, gameID)
	if err != nil {
		errorResponse(resp, http.StatusNotFound, err)
		return
	}

	sort.Slice(bag.Achievements.Achievements, func(i, j int) bool {
		return bag.Achievements.Achievements[i].GlobalPercentage > bag.Achievements.Achievements[j].GlobalPercentage
	})

	template := "game.gohtml"
	renderHtml(resp, http.StatusOK, template, bag)
}
