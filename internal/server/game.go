package server

import (
	"net/http"
	"strconv"

	"github.com/taiidani/achievements/internal/data"
)

type gameBag struct {
	baseBag
	UserID string
	Game   data.Game
}

func gameHandler(resp http.ResponseWriter, req *http.Request) {
	bag := gameBag{}
	bag.Page = "game"
	bag.UserID = req.URL.Query().Get("user-id")

	gameIDString := req.URL.Query().Get("game-id")
	gameID, _ := strconv.ParseUint(gameIDString, 10, 64)

	if bag.UserID != "" && gameID > 0 {
		// My userID is 76561197970932835
		game, err := data.GetGame(req.Context(), bag.UserID, gameID)
		if err != nil {
			errorResponse(resp, http.StatusNotFound, err)
			return
		}

		bag.Game = game
	}

	renderHtml(resp, http.StatusOK, "game.gohtml", bag)
}
