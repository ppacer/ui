package ui

import (
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/ppacer/core/api"
	"github.com/ppacer/core/scheduler"
)

const (
	dagrunDetailsErr     = "dagrunDetailsErr"
	dagrunTaskDetailsErr = "dagrunTaskDetailsErr"
	dagrunActionsErr     = "dagrunActionsErr"
	maxTaskIndent        = 10
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
	pdrd.cleanDagrunDetailsErr()
	runIdStr := r.PathValue("runId")
	runId, castErr := strconv.Atoi(runIdStr)
	if castErr != nil {
		pdrd.logger.Error("Invalid runId. Cannot cast to int.", "runId",
			runIdStr)
		pdrd.Errors[dagrunDetailsErr] =
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
		pdrd.Errors[dagrunDetailsErr] = msg
	}
	pdrd.Details = pdrd.prepareDagrunTaskDetails(drd, maxTaskIndent)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	renderErr := pdrd.templates.Render(w, "page_dagrun_details", pdrd)
	if renderErr != nil {
		pdrd.logger.Error("Cannot render <page_dagrun_details>", "err",
			renderErr.Error())
	}
}

// HTTP handler for restarting DAG run.
func (pdrd *pageDagRunDetails) RestartDagRunHandler(w http.ResponseWriter, r *http.Request) {
	pdrd.cleanDagrunActionsErr()
	r.ParseForm()

	dagId := r.FormValue("dagId")
	execTs := r.FormValue("execTs")
	runId := r.FormValue("runId")

	if dagId == "" || execTs == "" {
		pdrd.logger.Error("Invalid input for DAG restarting", "dagId", dagId,
			"execTs", execTs)
		pdrd.Errors[dagrunActionsErr] = "Cannot restart DAG run - invalid input"
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		renderErr := pdrd.templates.Render(w, "page_dagrun_details", pdrd)
		if renderErr != nil {
			pdrd.logger.Error("Cannot render <page_dagrun_details>", "err",
				renderErr.Error())
		}
		return
	}
	input := api.DagRunRestartInput{DagId: dagId, ExecTs: execTs}
	pdrd.logger.Info("Restarting DAG run", "input", input)

	err := pdrd.schedApi.RestartDagRun(input)
	if err != nil {
		pdrd.logger.Error("Error while restarting DAG run", "input", input,
			"err", err.Error())
		pdrd.Errors[dagrunActionsErr] = "Cannot restart DAG run"
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		renderErr := pdrd.templates.Render(w, "page_dagrun_details", pdrd)
		if renderErr != nil {
			pdrd.logger.Error("Cannot render <page_dagrun_details>", "err",
				renderErr.Error())
		}
		return
	}

	// Render DAG run summary once the DAG run is restarted.
	w.Header().Set("HX-Redirect", fmt.Sprintf("/dagruns/%s", runId))
	w.WriteHeader(http.StatusOK)
}

func (pdrd *pageDagRunDetails) RefreshSingleTaskDetailsHandler(
	w http.ResponseWriter, r *http.Request,
) {
	pdrd.cleanTaskDetailsErr()
	var detailsErr error
	var taskDetails api.UIDagrunTask

	runId, taskId, retry, taskPos, parseErr := parseTaskLogsArgs(r)
	if parseErr != nil {
		pdrd.logger.Error("Invalid path arguments for RefreshSingleTaskDetailsHandler",
			"parseErr", parseErr.Error())
		pdrd.Errors[dagrunTaskDetailsErr] =
			fmt.Sprintf("Invalid arguments for refreshing task details: %s",
				parseErr.Error())
	}

	if parseErr == nil {
		taskDetails, detailsErr = pdrd.schedApi.UIDagrunTaskDetails(
			runId, taskId, retry,
		)
		if detailsErr != nil {
			pdrd.logger.Error("Cannot get UI DAG run task details", "runId",
				runId, "taskId", taskId, "retry", retry, "err",
				detailsErr.Error())
			pdrd.Errors[dagrunTaskDetailsErr] = "cannot get DAG run task details"
		}
	}
	if parseErr != nil || detailsErr != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		renderErr := pdrd.templates.Render(w, "page_dagrun_details", pdrd)
		if renderErr != nil {
			pdrd.logger.Error("Cannot render <page_dagrun_details>", "err",
				renderErr.Error())
		}
		return
	}

	drt := DagrunTask{
		RunId:          int64(runId),
		TaskId:         taskId,
		Retry:          retry,
		InsertTs:       taskDetails.InsertTs,
		TaskNoStarted:  false,
		Status:         taskDetails.Status,
		Pos:            taskPos,
		Duration:       taskDetails.Duration,
		Config:         taskDetails.Config,
		TaskLogs:       toTaskLogs(taskDetails.TaskLogs),
		LogsWindowOpen: true,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	renderErr := pdrd.templates.Render(w, "dagrun_details_task_item", drt)
	if renderErr != nil {
		pdrd.logger.Error("Cannot render <dagrun_details_task_item>",
			"err", renderErr.Error())
	}
}

func parseTaskLogsArgs(r *http.Request) (int, string, int, TaskPos, error) {
	var taskPos TaskPos
	runId, parseRunIdErr := getPathValueInt(r, "runId")
	if parseRunIdErr != nil {
		err := fmt.Errorf("invalid runId: %w", parseRunIdErr)
		return -1, "", -1, taskPos, err
	}
	taskId, parseTaskIdErr := getPathValueStr(r, "taskId")
	if parseTaskIdErr != nil {
		err := fmt.Errorf("invalid taskId: %w", parseTaskIdErr)
		return -1, "", -1, taskPos, err
	}
	retry, parseRetryErr := getPathValueInt(r, "retry")
	if parseRetryErr != nil {
		err := fmt.Errorf("invalid retry argument: %w", parseRetryErr)
		return -1, "", -1, taskPos, err
	}

	taskPosStr, tpErr := getPathValueStr(r, "taskPos")
	if tpErr != nil {
		err := fmt.Errorf("invalid taskPos argument: %w", tpErr)
		return -1, "", -1, taskPos, err
	}
	const taskPosFields = 3
	taskPosSplit := strings.Split(taskPosStr, "_")
	if len(taskPosSplit) != taskPosFields {
		err := fmt.Errorf("invalid taskPos value, expected %d_%d_%d format")
		return -1, "", -1, taskPos, err
	}
	var posValues [taskPosFields]int
	for i := 0; i < taskPosFields; i++ {
		num, castErr := strconv.Atoi(taskPosSplit[i])
		if castErr != nil {
			err := fmt.Errorf("invalid taskPos value, cannot convert to integer: %w",
				castErr)
			return -1, "", -1, taskPos, err
		}
		posValues[i] = num
	}
	taskPos.Depth = posValues[0]
	taskPos.Width = posValues[1]
	taskPos.Indent = posValues[2]

	return runId, taskId, retry, taskPos, nil
}

func (pdrd *pageDagRunDetails) prepareDagrunTaskDetails(
	drd api.UIDagrunDetails, maxIndent int,
) DagrunDetails {
	return DagrunDetails{
		RunId:     drd.RunId,
		DagId:     drd.DagId,
		ExecTs:    drd.ExecTs,
		ExecTsRaw: drd.ExecTsRaw,
		Status:    drd.Status,
		Duration:  drd.Duration,
		Tasks:     prepareDagrunTasks(drd.RunId, drd.Tasks, maxIndent),
	}
}

func (pdrd *pageDagRunDetails) cleanDagrunDetailsErr() {
	if _, exist := pdrd.Errors[dagrunDetailsErr]; exist {
		pdrd.Errors[dagrunDetailsErr] = ""
	}
}

func (pdrd *pageDagRunDetails) cleanTaskDetailsErr() {
	if _, exist := pdrd.Errors[dagrunTaskDetailsErr]; exist {
		pdrd.Errors[dagrunTaskDetailsErr] = ""
	}
}

func (pdrd *pageDagRunDetails) cleanDagrunActionsErr() {
	if _, exist := pdrd.Errors[dagrunActionsErr]; exist {
		pdrd.Errors[dagrunActionsErr] = ""
	}
}

func prepareDagrunTasks(runId int64, tasks []api.UIDagrunTask, maxIndent int) []DagrunTask {
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].Pos.Depth != tasks[j].Pos.Depth {
			return tasks[i].Pos.Depth < tasks[j].Pos.Depth
		}
		if tasks[i].Pos.Width != tasks[j].Pos.Width {
			return tasks[i].Pos.Width < tasks[j].Pos.Width
		}
		return tasks[i].Retry < tasks[j].Retry
	})
	result := make([]DagrunTask, len(tasks))

	for i := 0; i < len(tasks); i++ {
		indent := tasks[i].Pos.Depth
		if indent > maxIndent {
			indent = maxIndent
		}
		drt := DagrunTask{
			RunId:         runId,
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
			Duration:       tasks[i].Duration,
			Config:         tasks[i].Config,
			TaskLogs:       toTaskLogs(tasks[i].TaskLogs),
			LogsWindowOpen: false,
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
	RunId     int64
	DagId     string
	ExecTs    api.Timestamp
	ExecTsRaw string
	Status    string
	Duration  string
	Tasks     []DagrunTask
}

type DagrunTask struct {
	RunId          int64
	TaskId         string
	Retry          int
	InsertTs       api.Timestamp
	TaskNoStarted  bool
	Status         string
	Pos            TaskPos
	Duration       string
	Config         string
	TaskLogs       TaskLogs
	LogsWindowOpen bool
	Errors         map[string]string
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
