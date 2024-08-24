package ui

import (
	"log/slog"
	"net/http"

	"github.com/ppacer/core/scheduler"
)

type pageDags struct {
	Page    string
	Version string

	templates *templates
	schedApi  scheduler.API
	logger    *slog.Logger
}

func newPageDags(
	schedApi scheduler.API, tmpl *templates, logger *slog.Logger,
	config Config,
) *pageDags {
	if logger == nil {
		logger = defaultLogger()
	}
	return &pageDags{
		Page:      "DAGs",
		Version:   Version,
		templates: tmpl,
		schedApi:  schedApi,
		logger:    logger,
	}
}

// Main handler for "DAGs" page.
func (pd *pageDags) MainHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	renderErr := pd.templates.Render(w, "page_dags", pd)
	if renderErr != nil {
		pd.logger.Error("Cannot render <page_dags>", "err",
			renderErr.Error())
		// TODO handle errors on UI
	}
}
