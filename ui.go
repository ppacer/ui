package ui

import (
	"embed"
	"io"
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
	config       Config
}

// NewUI creates new instance of ppacer UI.
func NewUI(schedulerUrl string, logger *slog.Logger, config *Config) *UI {
	if logger == nil {
		logger = defaultLogger()
	}
	if config == nil {
		cnf := DefaultConfig
		config = &cnf
	}
	cfg := scheduler.DefaultClientConfig
	return &UI{
		logger:       logger,
		schedulerAPI: scheduler.NewClient(schedulerUrl, nil, logger, cfg),
		config:       *config,
	}
}

// NewUIWithMocks creates new instance of ppacer UI which uses mocked Scheduler
// in stead of actual ppacer Scheduler. It's meant primarily for local
// development.
func NewUIWithMocks(logger *slog.Logger, config *Config) *UI {
	if config == nil {
		cnf := DefaultConfig
		config = &cnf
	}
	return &UI{
		logger:       logger,
		schedulerAPI: SchedulerMock{},
		config:       *config,
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
	dagruns := newPageDagRuns(s.schedulerAPI, templates, s.logger, s.config)
	mux.HandleFunc("/", dagruns.MainHandler)
	mux.HandleFunc("GET /dagruns/stats", dagruns.StatsHandler)
	mux.HandleFunc("GET /dagruns/latest", dagruns.ListHandler)
	mux.HandleFunc("POST /dagruns/latest/len", dagruns.UpdateDagRunsNumHandler)
	mux.HandleFunc("POST /dagruns/sync/stop", dagruns.SetSyncSeconds(1000000))
	mux.HandleFunc("POST /dagruns/sync/start", dagruns.SetSyncSeconds(1))

	// Page for DAG run details for given runId
	drDetails := newPageDagRunDetails(s.schedulerAPI, templates, s.logger)
	mux.HandleFunc("/dagruns/{runId}", drDetails.MainHandler)

	// Page for DAGs
	dagsPage := newPageDags(s.schedulerAPI, templates, s.logger, s.config)
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
