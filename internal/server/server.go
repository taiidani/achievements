package server

import (
	"embed"
	"html/template"
	"log/slog"
	"net/http"
)

type Server struct {
	*http.Server
}

//go:embed templates
var templates embed.FS

// devMode can be toggled to pull rendered files from the filesystem or the embedded FS.
var devMode = false

func NewServer() *Server {
	mux := http.NewServeMux()

	srv := &Server{
		Server: &http.Server{
			Addr:    ":http",
			Handler: mux,
		},
	}
	srv.addRoutes(mux)

	return srv
}

func (s *Server) addRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/game", gameHandler)
	mux.HandleFunc("/about", aboutHandler)
	mux.HandleFunc("/assets/*", assetsHandler)
}

func renderHtml(writer http.ResponseWriter, code int, file string, data any) {
	log := slog.With("name", file, "code", code)

	var t *template.Template
	var err error
	if devMode {
		t, err = template.ParseGlob("internal/server/templates/**")
	} else {
		t, err = template.ParseFS(templates, "templates/**")
	}
	if err != nil {
		log.Error("Could not parse templates", "error", err)
		return
	}

	log.Debug("Rendering file")
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

	renderHtml(writer, code, "error.gohtml", data)
}
