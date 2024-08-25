package server

import (
	"net/http"
)

type aboutBag struct {
	baseBag
}

func (s *Server) aboutHandler(resp http.ResponseWriter, req *http.Request) {
	data := aboutBag{}
	data.Page = "about"

	renderHtml(resp, http.StatusOK, "about.gohtml", data)
}
