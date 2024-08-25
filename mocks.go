package ui

import (
	"errors"
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

// TODO
func (sm SchedulerMock) UIDagrunDetails(runId int) (api.UIDagrunDetails, error) {
	// TODO
	return api.UIDagrunDetails{}, errors.New("not implemented")
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
