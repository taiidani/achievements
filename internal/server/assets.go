package server

import (
	"embed"
	"log/slog"
	"net/http"
	"path/filepath"
)

//go:embed assets
var assets embed.FS

func (s *Server) assetsHandler(resp http.ResponseWriter, req *http.Request) {
	slog.Debug("Serving file", "path", req.URL.Path)
	if DevMode {
		http.ServeFile(resp, req, filepath.Join("internal", "server", req.URL.Path))
	} else {
		http.ServeFileFS(resp, req, assets, req.URL.Path)
	}
}
