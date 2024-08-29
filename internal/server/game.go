package server

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/taiidani/achievements/internal/data"
)

type gameBag struct {
	baseBag
	UserID       string
	Game         data.Game
	Achievements data.Achievements
}

func (s *Server) gameHandler(resp http.ResponseWriter, req *http.Request) {
	bag := gameBag{}
	bag.Page = "game"
	bag.UserID = req.URL.Query().Get("user-id")

	gameIDString := req.URL.Query().Get("game-id")
	gameID, _ := strconv.ParseUint(gameIDString, 10, 64)

	if bag.UserID != "" && gameID > 0 {
		// My userID is 76561197970932835
		game, err := s.backend.GetGame(req.Context(), bag.UserID, gameID)
		if err != nil {
			errorResponse(resp, http.StatusNotFound, err)
			return
		}
		bag.Game = game

		bag.Achievements, err = s.backend.GetAchievements(req.Context(), bag.UserID, gameID)
		if err != nil {
			errorResponse(resp, http.StatusNotFound, err)
			return
		}

		sort.Slice(bag.Achievements.Achievements, func(i, j int) bool {
			return bag.Achievements.Achievements[i].GlobalPercentage > bag.Achievements.Achievements[j].GlobalPercentage
		})
	}

	template := "game.gohtml"
	if req.Header.Get("HX-Request") != "" {
		template = "game-body.gohtml"
	}

	renderHtml(resp, http.StatusOK, template, bag)
}
