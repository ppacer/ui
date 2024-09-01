package ui

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/ppacer/core/api"
	"github.com/ppacer/core/dag"
	"github.com/ppacer/core/scheduler"
	"github.com/ppacer/core/timeutils"
)

// SchedulerMock mocks ppacer scheduler.API, so the UI can be run without
// actual ppacer Scheduler running.
type SchedulerMock struct {
}

// GetTask return random TaskToExec.
func (sm SchedulerMock) GetTask() (api.TaskToExec, error) {
	const dagId = "mock_dag"
	r := rand.Intn(100) + 1

	tte := api.TaskToExec{
		DagId:  dagId,
		ExecTs: timeutils.ToStringUI(time.Now()),
		TaskId: fmt.Sprintf("task_%d", r),
		Retry:  0,
	}

	return tte, nil
}

// UpsertTaskStatus does nothing and return nil error every time.
func (sm SchedulerMock) UpsertTaskStatus(
	tte api.TaskToExec, status dag.TaskStatus, taskErr error,
) error {
	return nil
}

// GetState always returns RUNNING state.
func (sm SchedulerMock) GetState() (scheduler.State, error) {
	return scheduler.StateRunning, nil
}

// UIDagrunStats returns random stats on DAG runs.
func (sm SchedulerMock) UIDagrunStats() (api.UIDagrunStats, error) {
	return api.UIDagrunStats{
		Dagruns:               randomStatusCounts(250),
		DagrunTasks:           randomStatusCounts(1000),
		DagrunQueueLen:        rand.Intn(50),
		TaskSchedulerQueueLen: rand.Intn(200),
		GoroutinesNum:         rand.Intn(1000) + 5,
	}, nil
}

// UIDagrunLatest returns a slice of n random DAG runs.
func (sm SchedulerMock) UIDagrunLatest(n int) (api.UIDagrunList, error) {
	runId := rand.Intn(1000) + 11
	dagIds := []string{
		"sample_dag",
		"mock_dag",
		"sample_mock_longer_name_dag",
	}

	list := make(api.UIDagrunList, n)
	for i := 0; i < n; i++ {
		id := rand.Intn(len(dagIds))
		list[i] = randomDagrunRow(runId+i, dagIds[id])
	}
	return list, nil
}

// UIDagrunDetails returns random data on given DAG run details.
func (sm SchedulerMock) UIDagrunDetails(runId int) (api.UIDagrunDetails, error) {
	now := time.Now()
	end := now.Add(time.Duration(rand.Intn(1000)) * time.Millisecond)
	dagIds := []string{
		"sample_dag",
		"sample_mock_longer_name_dag",
		"linked_list",
		"complex_dag",
	}
	dagId := dagIds[rand.Intn(len(dagIds))]
	drd := api.UIDagrunDetails{
		RunId:    int64(runId),
		DagId:    dagId,
		ExecTs:   api.ToTimestamp(now),
		Status:   randomStatus(),
		Duration: end.Sub(now).String(),
		Tasks:    randomDagrunTasks(dagId),
	}
	return drd, nil
}

func randomDagrunTasks(dagId string) []api.UIDagrunTask {
	length := rand.Intn(10) + 3
	switch dagId {
	case "sample_dag":
		return randomDagrunTasksSampleDag()
	case "linked_list":
		return randomDagrunTasksLinkedList(length, rand.Intn(length+1))
	default:
	}
	return []api.UIDagrunTask{}
}

func randomDagrunTasksSampleDag() []api.UIDagrunTask {
	startTs := time.Now()
	d2l := rand.Intn(5) + 1
	tasks := make([]api.UIDagrunTask, 0, d2l+2)

	start := api.UIDagrunTask{
		TaskId:        "start",
		Retry:         0,
		InsertTs:      api.ToTimestamp(time.Now()),
		TaskNoStarted: false,
		Pos:           api.TaskPos{Depth: 1, Width: 1},
		Status:        dag.TaskSuccess.String(),
		Duration:      "1.015ms",
		Config:        "",
		TaskLogs:      api.UITaskLogs{},
	}
	finish := api.UIDagrunTask{
		TaskId:        "finish",
		Retry:         0,
		InsertTs:      api.ToTimestamp(time.Now()),
		TaskNoStarted: true,
		Pos:           api.TaskPos{Depth: 3, Width: 1},
		Status:        dag.TaskNoStatus.String(),
		Duration:      "",
		Config:        "",
		TaskLogs:      api.UITaskLogs{},
	}
	tasks = append(tasks, start)

	tasksNotStarted := false
	for i := 0; i < d2l; i++ {
		if i > 2 {
			tasksNotStarted = true
		}
		t := api.UIDagrunTask{
			TaskId:        fmt.Sprintf("task_2%d", i+1),
			Retry:         0,
			InsertTs:      api.ToTimestamp(time.Now()),
			TaskNoStarted: tasksNotStarted,
			Pos:           api.TaskPos{Depth: 2, Width: i + 1},
			Status:        dag.TaskNoStatus.String(),
		}
		if !tasksNotStarted {
			t.Status = randomStatus()
			taskEnd := startTs.Add(time.Duration(rand.Intn(10000)) * time.Millisecond)
			t.Duration = taskEnd.Sub(startTs).String()
			t.Config = `{X:10,Y:"value"}`
			t.TaskLogs = randomTaskLogs(rand.Intn(5))
		}
		tasks = append(tasks, t)
	}
	tasks = append(tasks, finish)
	return tasks
}

func randomDagrunTasksLinkedList(length int, tasksDone int) []api.UIDagrunTask {
	start := time.Now()
	tasks := make([]api.UIDagrunTask, 0, length)
	tasksNotStarted := false

	for i := 0; i < length; i++ {
		if i == tasksDone {
			tasksNotStarted = true
		}
		task := api.UIDagrunTask{
			TaskId:        fmt.Sprintf("task_%d", i+1),
			Retry:         0,
			InsertTs:      api.ToTimestamp(time.Now()),
			TaskNoStarted: tasksNotStarted,
			Pos:           api.TaskPos{Depth: i, Width: 1},
			Status:        dag.TaskNoStatus.String(),
		}
		if !tasksNotStarted {
			task.Status = randomStatus()
			taskEnd := start.Add(time.Duration(rand.Intn(10000)) * time.Millisecond)
			task.Duration = taskEnd.Sub(start).String()
			task.Config = `{X:10,Y:"value"}`
			task.TaskLogs = randomTaskLogs(rand.Intn(15))
		}
		tasks = append(tasks, task)
	}
	return tasks
}

func randomTaskLogs(length int) api.UITaskLogs {
	records := make([]api.UITaskLogRecord, 0, length)

	for i := 0; i < length; i++ {
		record := api.UITaskLogRecord{
			InsertTs:       api.ToTimestamp(time.Now()),
			Level:          randomLogLevel(),
			Message:        randomString(10, 200),
			AttributesJson: randomLogAttr(),
		}
		records = append(records, record)
	}

	return api.UITaskLogs{
		LogRecordsCount: length,
		LoadedRecords:   length,
		Records:         records,
	}
}

func randomStatusCounts(interval int) api.StatusCounts {
	return api.StatusCounts{
		Success:   rand.Intn(interval+1) + interval/4,
		Failed:    rand.Intn((interval + 1) / 10),
		Scheduled: rand.Intn(3),
		Running:   rand.Intn((interval + 1) / 7),
	}
}

func randomDagrunRow(runId int, dagId string) api.UIDagrunRow {
	tasks := rand.Intn(100)
	now := time.Now().In(randomLocation())
	end := now.Add(time.Duration(rand.Intn(1000)) * time.Millisecond)

	return api.UIDagrunRow{
		RunId:            int64(runId),
		DagId:            dagId,
		ExecTs:           api.ToTimestamp(now),
		InsertTs:         api.ToTimestamp(now.Add(3 * time.Millisecond)),
		Status:           randomStatus(),
		StatusUpdateTs:   api.ToTimestamp(now.Add(3 * time.Second)),
		Duration:         end.Sub(now).String(),
		TaskNum:          tasks,
		TaskCompletedNum: tasks - rand.Intn(tasks+1),
	}
}

func randomLocation() *time.Location {
	if rand.Intn(10) > 4 {
		return time.UTC
	}
	return time.Local
}

func randomStatus() string {
	r := rand.Intn(10)
	if r > 8 {
		return dag.RunFailed.String()
	} else if r > 7 {
		return dag.RunSuccess.String()
	}
	return dag.RunRunning.String()
}

func randomLogLevel() string {
	r := rand.Intn(10)
	if r > 8 {
		return "ERROR"
	}
	if r > 6 {
		return "WARN"
	}
	return "INFO"
}

const charset = `                 abcdefghij
klmnopqrstuvw       xyzABCDEFGH       IJKLMNOPQR  STUVWXYZ0123456789"
`

func randomString(minLen, maxLen int) string {
	l := rand.Intn(maxLen-minLen+1) + minLen
	word := make([]byte, l)
	for i := 0; i < l; i++ {
		word[i] = charset[rand.Intn(len(charset))]
	}
	return string(word)
}

func randomLogAttr() string {
	l := rand.Intn(3)
	attr := make(map[string]any)
	for i := 0; i < l; i++ {
		if i%2 == 0 {
			attr[randomString(1, 10)] = rand.Intn(100)
		} else {
			attr[randomString(1, 10)] = randomString(10, 30)
		}
	}
	json, _ := json.Marshal(attr)
	return string(json)
}
