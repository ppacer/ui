package ui

import (
	"embed"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"text/template"

	"github.com/ppacer/core/scheduler"
)

//go:embed views/*.html
var viewsFS embed.FS

//go:embed assets/* css/*
var staticFS embed.FS

// UI represents ppacer UI.
type UI struct {
	logger       *slog.Logger
	schedulerAPI scheduler.API
}

// NewUI creates new instance of ppacer UI.
func NewUI(schedulerUrl string, logger *slog.Logger) *UI {
	if logger == nil {
		logger = defaultLogger()
	}
	cfg := scheduler.DefaultClientConfig
	return &UI{
		logger:       logger,
		schedulerAPI: scheduler.NewClient(schedulerUrl, nil, logger, cfg),
	}
}

// NewUIWithMocks creates new instance of ppacer UI which uses mocked Scheduler
// in stead of actual ppacer Scheduler. It's meant primarily for local
// development.
func NewUIWithMocks(logger *slog.Logger) *UI {
	return &UI{
		logger:       logger,
		schedulerAPI: SchedulerMock{},
	}
}

// DefaultStarted starts HTTP server which serves ppacer UI in default
// configuration. This function is meant to reduce boilerplate for simple
// examples and tests. When there is an error on starting UI server this
// function panics.
func DefaultStarted(schedulerPort, uiPort int) {
	schedulerUrl := fmt.Sprintf("http://localhost:%d", schedulerPort)
	uiDefault := NewUI(schedulerUrl, defaultLogger())
	portStr := fmt.Sprintf(":%d", uiPort)
	fmt.Println("Starting ppacer UI on ", portStr)
	err := http.ListenAndServe(portStr, uiDefault.Server())
	if err != nil {
		log.Panicf("Cannot start ppacer UI server: %s", err.Error())
	}
}

// DefaultStartedMocks starts HTTP server which serves ppacer UI in default
// configuration. Similarly to DefaultStarted, but instead of communicating
// with actual ppacer Scheduler it would used mocked data within the UI server.
// This function is primarily for local development convenience. When there is
// an error on starting UI server this function panics.
func DefaultStartedMocks(uiPort int) {
	uiDefault := NewUIWithMocks(defaultLogger())
	portStr := fmt.Sprintf(":%d", uiPort)
	fmt.Println("Starting ppacer UI with mocked data on ", portStr)
	err := http.ListenAndServe(portStr, uiDefault.Server())
	if err != nil {
		log.Panicf("Cannot start ppacer UI server: %s", err.Error())
	}
}

// Server set ups ppacer UI server which serves the web UI and provides
// necessary endpoints for communicating with ppacer Scheduler.
func (s *UI) Server() http.Handler {
	mux := http.NewServeMux()
	templates := newTemplates()

	// Serve static files from embedded filesystem
	mux.Handle("/assets/", http.FileServer(http.FS(staticFS)))
	mux.Handle("/css/", http.FileServer(http.FS(staticFS)))

	// Page for DAG runs (main)
	dagruns := newPageDagRuns(s.schedulerAPI, templates, s.logger)
	mux.HandleFunc("/", dagruns.MainHandler)
	mux.HandleFunc("GET /dagruns/stats", dagruns.StatsHandler)
	mux.HandleFunc("GET /dagruns/latest", dagruns.ListHandler)
	mux.HandleFunc("POST /dagruns/latest/len", dagruns.UpdateDagRunsNumHandler)
	mux.HandleFunc("POST /dagruns/sync/stop", dagruns.SetSyncSeconds(1000000))
	mux.HandleFunc("POST /dagruns/sync/start", dagruns.SetSyncSeconds(1))

	// Page for DAGs
	dagsPage := newPageDags(s.schedulerAPI, templates, s.logger)
	mux.HandleFunc("/dags", dagsPage.MainHandler)

	return mux
}

type templates struct {
	templates *template.Template
}

func (t *templates) Render(w io.Writer, name string, data any) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func newTemplates() *templates {
	return &templates{
		templates: template.Must(template.ParseFS(viewsFS, "views/*.html")),
	}
}
