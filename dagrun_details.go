package ui

import (
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"

	"github.com/ppacer/core/api"
	"github.com/ppacer/core/scheduler"
)

const (
	dagrunDetailsRundIdErr = "dagrunDetailsRunIdErr"
)

// Type pageDagRunDetails keeps data required for DAG run details
// (/dagruns/{runId}) page.
type pageDagRunDetails struct {
	Page    string
	Details DagrunDetails
	Errors  map[string]string
	Version string

	templates *templates
	schedApi  scheduler.API
	logger    *slog.Logger
}

// newPageDagRunDetails initialize new state for DAG run details page.
func newPageDagRunDetails(
	schedApi scheduler.API, tmpl *templates, logger *slog.Logger,
) *pageDagRunDetails {
	if logger == nil {
		logger = defaultLogger()
	}
	return &pageDagRunDetails{
		Page:    "Runs",
		Errors:  map[string]string{},
		Version: Version,

		templates: tmpl,
		schedApi:  schedApi,
		logger:    logger,
	}
}

// MainHandler prepares and renders
func (pdrd *pageDagRunDetails) MainHandler(w http.ResponseWriter, r *http.Request) {
	runIdStr := r.PathValue("runId")
	runId, castErr := strconv.Atoi(runIdStr)
	if castErr != nil {
		pdrd.logger.Error("Invalid runId. Cannot cast to int.", "runId",
			runIdStr)
		pdrd.Errors[dagrunDetailsRundIdErr] =
			fmt.Sprintf("Invalid runId (%s) - cannot cast it to integer",
				runIdStr)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		renderErr := pdrd.templates.Render(w, "page_dagrun_details", pdrd)
		if renderErr != nil {
			pdrd.logger.Error("Cannot render <page_dagrun_details>", "err",
				renderErr.Error())
		}
		return
	}

	drd, err := pdrd.schedApi.UIDagrunDetails(runId)
	if err != nil {
		msg := fmt.Sprintf("cannot read DAG run details: %s", err.Error())
		http.Error(w, msg, http.StatusInternalServerError)
	}
	pdrd.Details = pdrd.prepareDagrunTaskDetails(drd, 10)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	renderErr := pdrd.templates.Render(w, "page_dagrun_details", pdrd)
	if renderErr != nil {
		pdrd.logger.Error("Cannot render <page_dagrun_details>", "err",
			renderErr.Error())
	}
}

func (pdrd *pageDagRunDetails) prepareDagrunTaskDetails(
	drd api.UIDagrunDetails, maxIndent int,
) DagrunDetails {
	return DagrunDetails{
		RunId:    drd.RunId,
		DagId:    drd.DagId,
		ExecTs:   drd.ExecTs,
		Status:   drd.Status,
		Duration: drd.Duration,
		Tasks:    prepareDagrunTasks(drd.Tasks, maxIndent),
	}
}

func prepareDagrunTasks(tasks []api.UIDagrunTask, maxIndent int) []DagrunTask {
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].Pos.Depth != tasks[j].Pos.Depth {
			return tasks[i].Pos.Depth < tasks[j].Pos.Depth
		}
		return tasks[i].Pos.Width < tasks[j].Pos.Width
	})
	result := make([]DagrunTask, len(tasks))

	for i := 0; i < len(tasks); i++ {
		indent := tasks[i].Pos.Depth
		if indent > maxIndent {
			indent = maxIndent
		}
		drt := DagrunTask{
			TaskId:        tasks[i].TaskId,
			Retry:         tasks[i].Retry,
			InsertTs:      tasks[i].InsertTs,
			TaskNoStarted: tasks[i].TaskNoStarted,
			Status:        tasks[i].Status,
			Pos: TaskPos{
				Depth:  tasks[i].Pos.Depth,
				Width:  tasks[i].Pos.Width,
				Indent: indent,
			},
			Duration: tasks[i].Duration,
			Config:   tasks[i].Config,
			TaskLogs: toTaskLogs(tasks[i].TaskLogs),
		}
		result[i] = drt
	}
	return result
}

func toTaskLogs(tl api.UITaskLogs) TaskLogs {
	return TaskLogs{
		LogRecordsCount: tl.LogRecordsCount,
		LoadedRecords:   tl.LoadedRecords,
		Records:         toTaskLogRecords(tl.Records),
	}
}

func toTaskLogRecords(tlr []api.UITaskLogRecord) []TaskLogRecord {
	newTlr := make([]TaskLogRecord, len(tlr))
	for i := 0; i < len(tlr); i++ {
		newTlr[i] = TaskLogRecord{
			InsertTs:       tlr[i].InsertTs,
			Level:          tlr[i].Level,
			Message:        tlr[i].Message,
			AttributesJson: tlr[i].AttributesJson,
		}
	}
	return newTlr
}

type DagrunDetails struct {
	RunId    int64
	DagId    string
	ExecTs   api.Timestamp
	Status   string
	Duration string
	Tasks    []DagrunTask
}

type DagrunTask struct {
	TaskId        string
	Retry         int
	InsertTs      api.Timestamp
	TaskNoStarted bool
	Status        string
	Pos           TaskPos
	Duration      string
	Config        string
	TaskLogs      TaskLogs
}

// TaskPos represents a Task position in a DAG. Root starts in (D=1,W=1).
type TaskPos struct {
	Depth  int
	Width  int
	Indent int
}

// UITaskLogs represents information on DAG run task logs. By default read only
// fixed number of log records, to limit the request body size and on demand
// more log records can be loaded in a separate call.
type TaskLogs struct {
	LogRecordsCount int
	LoadedRecords   int
	Records         []TaskLogRecord
}

// UITaskLogRecord represents task log records with assumed DAG run and task
// information.
type TaskLogRecord struct {
	InsertTs       api.Timestamp
	Level          string
	Message        string
	AttributesJson string
}
