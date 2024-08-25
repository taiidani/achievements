package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/taiidani/achievements/internal/data"
)

type hxGameRowBag struct {
	UserID string
	GameID uint64
	Game   data.Game
}

func (s *Server) hxGameRowHandler(resp http.ResponseWriter, req *http.Request) {
	bag := hxGameRowBag{}
	bag.UserID = req.URL.Query().Get("user-id")

	gameIDString := req.URL.Query().Get("game-id")
	bag.GameID, _ = strconv.ParseUint(gameIDString, 10, 64)

	if bag.UserID == "" {
		errorResponse(resp, http.StatusBadRequest, fmt.Errorf("user ID is required"))
		return
	}

	game, err := s.backend.GetGame(req.Context(), bag.UserID, bag.GameID)
	if err != nil {
		errorResponse(resp, http.StatusNotFound, err)
		return
	}
	bag.Game = game

	renderHtml(resp, http.StatusOK, "index-row.gohtml", bag)
}
