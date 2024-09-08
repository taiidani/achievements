package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/taiidani/achievements/internal/data"
)

type hxGameRowBag struct {
	baseBag
	GameID       uint64
	Achievements data.Achievements
}

func (s *Server) hxGameRowHandler(resp http.ResponseWriter, req *http.Request) {
	bag := hxGameRowBag{baseBag: newBag(req, "")}

	gameIDString := req.URL.Query().Get("game-id")
	bag.GameID, _ = strconv.ParseUint(gameIDString, 10, 64)

	if bag.SteamID == "" {
		errorResponse(resp, http.StatusBadRequest, fmt.Errorf("user ID is required"))
		return
	}

	achievements, err := s.backend.GetAchievements(req.Context(), bag.SteamID, bag.GameID)
	if err != nil {
		errorResponse(resp, http.StatusNotFound, err)
		return
	}
	bag.Achievements = achievements

	renderHtml(resp, http.StatusOK, "hx-achievement-progress.gohtml", bag)
}
