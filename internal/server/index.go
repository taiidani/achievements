package server

import (
	"net/http"
	"sort"

	"github.com/taiidani/achievements/internal/data"
)

type indexBag struct {
	baseBag
	UserID string
	User   data.User
	Games  []data.Game
}

func indexHandler(resp http.ResponseWriter, req *http.Request) {
	bag := indexBag{}
	bag.Page = "home"
	bag.UserID = req.URL.Query().Get("user-id")

	if bag.UserID != "" {
		// My userID is 76561197970932835
		user, err := data.GetUser(req.Context(), bag.UserID)
		if err != nil {
			errorResponse(resp, http.StatusNotFound, err)
			return
		}
		bag.User = user

		games, err := data.GetGames(req.Context(), bag.UserID)
		if err != nil {
			errorResponse(resp, http.StatusNotFound, err)
			return
		}

		bag.Games = games
	}

	sort.Slice(bag.Games, func(i, j int) bool {
		return bag.Games[i].DisplayName < bag.Games[j].DisplayName
	})

	renderHtml(resp, http.StatusOK, "index.gohtml", bag)
}
