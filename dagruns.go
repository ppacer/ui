package ui

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/ppacer/core/api"
	"github.com/ppacer/core/scheduler"
)

const (
	dagrunStatsErrorKey = "dagrunStatsErr"
	dagrunListErrorKey  = "dagrunListErr"
)

type pageDagRuns struct {
	Page          string
	Stats         api.UIDagrunStats
	LatestDagRuns api.UIDagrunList
	DagRunsNum    int
	SyncSeconds   int
	Errors        map[string]string

	templates *templates
	schedApi  scheduler.API
	logger    *slog.Logger
}

func newPageDagRuns(
	schedApi scheduler.API, tmpl *templates, logger *slog.Logger,
) *pageDagRuns {
	if logger == nil {
		logger = defaultLogger()
	}
	return &pageDagRuns{
		Page:        "Runs",
		DagRunsNum:  10,
		SyncSeconds: 1,
		Errors:      map[string]string{},

		templates: tmpl,
		schedApi:  schedApi,
		logger:    logger,
	}
}

// Main handler for "Runs" page.
func (pdr *pageDagRuns) MainHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	pdr.cleanStatsErr()
	pdr.cleanListErr()
	statsErr := pdr.syncCurrentStats()
	if statsErr != nil {
		msg := "Error while getting current DAG runs stats"
		pdr.logger.Error(msg, "err", statsErr.Error())
		pdr.Errors[dagrunStatsErrorKey] = fmt.Sprintf("%s: %s", msg,
			statsErr.Error())
	}
	listErr := pdr.syncLatestDagRuns()
	if listErr != nil {
		msg := "Error while getting latest DAG runs"
		pdr.logger.Error(msg, "n", pdr.DagRunsNum, "err", listErr.Error())
		pdr.Errors[dagrunListErrorKey] = fmt.Sprintf("%s: %s", msg,
			listErr.Error())
	}
	renderErr := pdr.templates.Render(w, "page_dagruns", pdr)
	if renderErr != nil {
		pdr.logger.Error("Cannot render <page_dagruns>", "err",
			renderErr.Error())
	}
}

// HTTP handler which refresh DAG runs statistics and render related component.
func (pdr *pageDagRuns) StatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	pdr.cleanStatsErr()
	statsErr := pdr.syncCurrentStats()
	if statsErr != nil {
		msg := "Error while getting current DAG runs stats"
		pdr.logger.Error(msg, "err", statsErr.Error())
		pdr.Errors[dagrunStatsErrorKey] = fmt.Sprintf("%s: %s", msg,
			statsErr.Error())
	}
	if err := pdr.templates.Render(w, "dagrun_stats", pdr); err != nil {
		pdr.logger.Error("Error while rendering <dagrun_stats>", "stats",
			pdr.Stats, "err", err.Error())
	}
}

func (pdr *pageDagRuns) UpdateDagRunsNumHandler(
	w http.ResponseWriter, r *http.Request,
) {
	if err := r.ParseForm(); err != nil {
		pdr.logger.Error("Cannot parse form with DagRunsNum", "err",
			err.Error())
		return
	}

	numStr := r.FormValue("num")
	num, err := strconv.Atoi(numStr)
	if err != nil {
		pdr.logger.Error("Cannot cast given value into number", "numStr",
			numStr, "err", err.Error())
		return
	}
	pdr.DagRunsNum = num
}

// SetSyncSeconds returns a HTTP handler for setting SyncSeconds.
func (pdr *pageDagRuns) SetSyncSeconds(seconds int) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		pdr.logger.Debug("Set SyncSeconds", "seconds", seconds)
		pdr.SyncSeconds = seconds
	}
}

// HTTP handler which refresh latest DAG runs list and render related component.
func (pdr *pageDagRuns) ListHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	pdr.cleanListErr()
	listErr := pdr.syncLatestDagRuns()
	if listErr != nil {
		msg := "Error while getting latest DAG runs list"
		pdr.logger.Error(msg, "n", pdr.DagRunsNum, "err", listErr.Error())
		pdr.Errors[dagrunListErrorKey] = fmt.Sprintf("%s: %s", msg,
			listErr.Error())
	}
	if err := pdr.templates.Render(w, "dagrun_list", pdr); err != nil {
		pdr.logger.Error("Error while rendering <dagrun_list>", "n",
			len(pdr.LatestDagRuns), "err", err.Error())
	}
}

func (pdr *pageDagRuns) syncCurrentStats() error {
	currentStats, err := pdr.schedApi.UIDagrunStats()
	if err != nil {
		return err
	}
	pdr.Stats = currentStats
	return nil
}

func (pdr *pageDagRuns) syncLatestDagRuns() error {
	dagruns, err := pdr.schedApi.UIDagrunLatest(pdr.DagRunsNum)
	if err != nil {
		return err
	}
	pdr.LatestDagRuns = dagruns
	return nil
}

func (pdr *pageDagRuns) cleanStatsErr() {
	if _, exist := pdr.Errors[dagrunStatsErrorKey]; exist {
		pdr.Errors[dagrunStatsErrorKey] = ""
	}
}

func (pdr *pageDagRuns) cleanListErr() {
	if _, exist := pdr.Errors[dagrunListErrorKey]; exist {
		pdr.Errors[dagrunListErrorKey] = ""
	}
}
