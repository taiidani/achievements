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
	backend   *data.Data
	publicURL string
	port      string
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

	publicURL := os.Getenv("PUBLIC_URL")
	if publicURL == "" {
		publicURL = "http://localhost:" + port
	}

	srv := &Server{
		Server: &http.Server{
			Addr:    fmt.Sprintf(":%s", port),
			Handler: mux,
		},
		publicURL: publicURL,
		port:      port,
		backend:   backend,
	}
	srv.addRoutes(mux)

	return srv
}

func (s *Server) addRoutes(mux *http.ServeMux) {
	mux.Handle("/", s.sessionMiddleware(http.HandlerFunc(s.indexHandler)))
	mux.Handle("/game/{id}", s.sessionMiddleware(http.HandlerFunc(s.gameHandler)))
	mux.Handle("/about", s.sessionMiddleware(http.HandlerFunc(s.aboutHandler)))
	mux.Handle("/assets/*", http.HandlerFunc(s.assetsHandler))
	mux.Handle("/hx/game/{id}/row", s.sessionMiddleware(http.HandlerFunc(s.hxGameRowHandler)))
	mux.Handle("/hx/game/{id}/pin", s.sessionMiddleware(http.HandlerFunc(s.hxGamePinHandler)))
	mux.Handle("/user/login", s.sessionMiddleware(http.HandlerFunc(s.userLoginHandler)))
	mux.Handle("/user/login/steam", s.sessionMiddleware(http.HandlerFunc(s.userLoginSteamHandler)))
	mux.Handle("/user/change", s.sessionMiddleware(http.HandlerFunc(s.userChangeHandler)))
	mux.Handle("/user/logout", http.HandlerFunc(s.userLogoutHandler))
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
	SessionKey string
	Session    *data.Session
	Page       string
	LoggedIn   bool
	SteamID    string
}

func (s *Server) newBag(r *http.Request, pageName string) baseBag {
	ret := baseBag{}
	ret.Page = pageName

	// Load the session if it exists
	cookie, err := r.Cookie("session")
	if err == nil {
		ret.SessionKey = cookie.Value
		sess, err := s.backend.GetSession(r.Context(), cookie.Value)
		if err != nil {
			slog.Warn("Unable to retrieve session", "key", cookie.Value, "error", err)
		} else {
			ret.Session = sess
			ret.LoggedIn = true
		}
	}

	// Prioritize the query parameter over the session ID
	ret.SteamID = r.FormValue("steam-id")
	if ret.SteamID == "" {
		ret.SteamID = r.Header.Get(steamIDHeaderKey)
	}

	return ret
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
