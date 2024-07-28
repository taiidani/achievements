package server

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/taiidani/achievements/internal/data"
	"github.com/taiidani/achievements/internal/steam"
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
			// Attempt to resolve the user's vanity URL into an ID
			client := steam.NewClient()
			vanity, err := client.ISteamUser.ResolveVanityURL(req.Context(), bag.UserID)
			if err != nil {
				errorResponse(resp, http.StatusNotFound, fmt.Errorf("could not resolve user id %q to a Steam User ID or Vanity URL", bag.UserID))
				return
			}

			bag.UserID = vanity.Response.SteamID
			user, err = data.GetUser(req.Context(), bag.UserID)
			if err != nil {
				errorResponse(resp, http.StatusNotFound, fmt.Errorf("could not resolve user id %q to a Steam User ID or Vanity URL", bag.UserID))
				return
			}
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
