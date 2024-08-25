package server

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/taiidani/achievements/internal/data"
)

type Server struct {
	backend *data.Data
	*http.Server
}

//go:embed templates
var templates embed.FS

// DevMode can be toggled to pull rendered files from the filesystem or the embedded FS.
var DevMode = os.Getenv("DEV") == "true"

func NewServer(backend *data.Data) *Server {
	mux := http.NewServeMux()

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("Required PORT environment variable not present")
	}

	srv := &Server{
		Server: &http.Server{
			Addr:    fmt.Sprintf(":%s", port),
			Handler: mux,
		},
		backend: backend,
	}
	srv.addRoutes(mux)

	return srv
}

func (s *Server) addRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", s.indexHandler)
	mux.HandleFunc("/game", s.gameHandler)
	mux.HandleFunc("/about", s.aboutHandler)
	mux.HandleFunc("/assets/*", s.assetsHandler)
	mux.HandleFunc("/hx/game/row", s.hxGameRowHandler)
}

func renderHtml(writer http.ResponseWriter, code int, file string, data any) {
	log := slog.With("name", file, "code", code)

	var t *template.Template
	var err error
	if DevMode {
		t, err = template.ParseGlob("internal/server/templates/**")
	} else {
		t, err = template.ParseFS(templates, "templates/**")
	}
	if err != nil {
		log.Error("Could not parse templates", "error", err)
		return
	}

	log.Debug("Rendering file", "dev", DevMode)
	writer.WriteHeader(code)
	err = t.ExecuteTemplate(writer, file, data)
	if err != nil {
		log.Error("Could not render template", "error", err)
	}
}

type baseBag struct {
	Page string
}

type errorBag struct {
	baseBag
	Message error
}

func errorResponse(writer http.ResponseWriter, code int, err error) {
	data := errorBag{
		Message: err,
	}

	slog.Error("Displaying error page", "error", err)
	renderHtml(writer, code, "error.gohtml", data)
}
