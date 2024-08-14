package main

//go:generate wit-bindgen tiny-go wit --out-dir=gen --gofmt

import (
	"fmt"

	. "github.com/wasmCloud/wasmcloud-contrib/demo/components/task-manager/gen"
)

type TaskManager struct{}

var _ ExportsWasmcloudTaskManagerTracker = &TaskManager{}

func init() {
	manager := &TaskManager{}
	SetExportsWasmcloudTaskManagerTracker(manager)
}

func main() {}

func (t *TaskManager) Start(category string, payload string) (result Result[string, ExportsWasmcloudTaskManagerTrackerOperationError]) {
	query := fmt.Sprintf("INSERT INTO tasks(payload, category) VALUES($1, '%s') RETURNING task_id", category)
	params := []WasmcloudPostgres0_1_0_draft_QueryPgValue{
		WasmcloudPostgres0_1_0_draft_TypesPgValueText(payload),
	}

	queryRes := WasmcloudPostgres0_1_0_draft_QueryQuery(query, params)
	if queryRes.IsErr() {
		WasiLoggingLoggingLog(WasiLoggingLoggingLevelError(), "Start", "Got an error creating task")
		result.SetErr(ExportsWasmcloudTaskManagerTrackerOperationError{Message: "Error creating task"})
		return
	}

	row := queryRes.Unwrap()

	if len(row) == 0 {
		WasiLoggingLoggingLog(WasiLoggingLoggingLevelError(), "Start", "Task doesn't exist")
		result.SetErr(ExportsWasmcloudTaskManagerTrackerOperationError{Message: "Task doesn't exist"})
		return
	}

	result.Set(row[0][0].Value.GetText())

	return
}

func (t *TaskManager) Fail(id string, outcome string) (result Result[struct{}, ExportsWasmcloudTaskManagerTrackerOperationError]) {
	// TODO(lxf): This is failing when more than one '$' is used
	query := fmt.Sprintf("UPDATE tasks SET completed_at = NOW(), err = $1 WHERE task_id = '%s'", id)
	params := []WasmcloudPostgres0_1_0_draft_QueryPgValue{
		WasmcloudPostgres0_1_0_draft_TypesPgValueText(outcome),
	}
	queryRes := WasmcloudPostgres0_1_0_draft_QueryQuery(query, params)
	if queryRes.IsErr() {
		WasiLoggingLoggingLog(WasiLoggingLoggingLevelError(), "Fail", "Got an error updating task")
		result.SetErr(ExportsWasmcloudTaskManagerTrackerOperationError{Message: "Error updating task"})
		return
	}

	result.Set(struct{}{})

	return
}

func (t *TaskManager) Complete(id string, outcome string) (result Result[struct{}, ExportsWasmcloudTaskManagerTrackerOperationError]) {
	// TODO(lxf): This is failing when more than one '$' is used
	prepareRes := WasmcloudPostgres0_1_0_draft_PreparedPrepare(fmt.Sprintf("UPDATE tasks SET completed_at = NOW(), result = $1 WHERE task_id = '%s'", id))
	if prepareRes.IsErr() {
		WasiLoggingLoggingLog(WasiLoggingLoggingLevelError(), "Complete", "Got an error creating prepared statement")
		result.SetErr(ExportsWasmcloudTaskManagerTrackerOperationError{Message: "Error creating prepared statement"})
		return
	}

	preparedStatement := prepareRes.Unwrap()

	params := []WasmcloudPostgres0_1_0_draft_QueryPgValue{
		WasmcloudPostgres0_1_0_draft_TypesPgValueText(outcome),
	}
	queryRes := WasmcloudPostgres0_1_0_draft_PreparedExec(preparedStatement, params)
	if queryRes.IsErr() {
		WasiLoggingLoggingLog(WasiLoggingLoggingLevelError(), "Complete", "Got an error updating task")
		result.SetErr(ExportsWasmcloudTaskManagerTrackerOperationError{Message: "Error updating task"})
		return
	}

	result.Set(struct{}{})

	return
}

func (t *TaskManager) Get(id string) (result Result[ExportsWasmcloudTaskManagerTrackerOperation, ExportsWasmcloudTaskManagerTrackerOperationError]) {
	query := "SELECT task_id,category,err,payload,result FROM tasks WHERE task_id = $1"
	params := []WasmcloudPostgres0_1_0_draft_QueryPgValue{
		WasmcloudPostgres0_1_0_draft_TypesPgValueText(id),
	}
	queryRes := WasmcloudPostgres0_1_0_draft_QueryQuery(query, params)
	if queryRes.IsErr() {
		WasiLoggingLoggingLog(WasiLoggingLoggingLevelError(), "Get", "Got an error from query")
		result.SetErr(ExportsWasmcloudTaskManagerTrackerOperationError{Message: "Error querying task"})
		return
	}

	row := queryRes.Unwrap()
	if len(row) == 0 {
		WasiLoggingLoggingLog(WasiLoggingLoggingLevelError(), "Get", "Task doesn't exist")
		result.SetErr(ExportsWasmcloudTaskManagerTrackerOperationError{Message: "Task doesn't exist"})
		return
	}

	taskResult := Option[string]{}
	if row[0][5].Value.Kind() != WasmcloudPostgres0_1_0_draft_TypesPgValueKindNull {
		taskResult.Set(row[0][5].Value.GetText())
	}

	taskFailure := Option[string]{}
	if row[0][3].Value.Kind() != WasmcloudPostgres0_1_0_draft_TypesPgValueKindNull {
		taskFailure.Set(row[0][3].Value.GetText())
	}
	result.Set(WasmcloudTaskManagerTypesOperation{
		Id:       row[0][0].Value.GetText(),
		Category: row[0][1].Value.GetText(),
		Payload:  row[0][4].Value.GetText(),
		Failure:  taskFailure,
		Result:   taskResult,
	})

	return
}

func formatTimestamp(t WasmcloudPostgres0_1_0_draft_TypesTimestamp) string {
	date := t.Date.GetYmd()
	return fmt.Sprintf("%02d-%02d-%02dT%02d:%02d:%02dZ", date.F0, date.F1, date.F2, t.Time.Hour, t.Time.Min, t.Time.Sec)
}

func (t *TaskManager) List(filter Option[string]) (result Result[[]ExportsWasmcloudTaskManagerTrackerOperation, ExportsWasmcloudTaskManagerTrackerOperationError]) {
	query := "SELECT task_id,category,payload,created_at,completed_at,result,err FROM tasks"
	var params []WasmcloudPostgres0_1_0_draft_QueryPgValue
	queryRes := WasmcloudPostgres0_1_0_draft_QueryQuery(query, params)
	if queryRes.IsErr() {
		WasiLoggingLoggingLog(WasiLoggingLoggingLevelError(), "List", "Got an error from query")
		result.SetErr(ExportsWasmcloudTaskManagerTrackerOperationError{Message: "Error listing tasks"})
		return
	}

	rows := queryRes.Unwrap()
	tasks := []ExportsWasmcloudTaskManagerTrackerOperation{}
	for _, row := range rows {
		taskResult := Option[string]{}
		taskFailure := Option[string]{}
		taskCompletedAt := Option[string]{}

		if row[4].Value.Kind() != WasmcloudPostgres0_1_0_draft_TypesPgValueKindNull {
			completedAt := row[4].Value.GetTimestamp()
			taskCompletedAt.Set(formatTimestamp(completedAt))
		}

		if row[5].Value.Kind() != WasmcloudPostgres0_1_0_draft_TypesPgValueKindNull {
			taskResult.Set(row[5].Value.GetText())
		}

		if row[6].Value.Kind() != WasmcloudPostgres0_1_0_draft_TypesPgValueKindNull {
			taskFailure.Set(row[6].Value.GetText())
		}

		createdAt := row[3].Value.GetTimestamp()

		tasks = append(tasks, WasmcloudTaskManagerTypesOperation{
			Id:        row[0].Value.GetText(),
			Category:  row[1].Value.GetText(),
			Payload:   row[2].Value.GetText(),
			CreatedAt: formatTimestamp(createdAt),

			Result:      taskResult,
			Failure:     taskFailure,
			CompletedAt: taskCompletedAt,
		})
	}

	result.Set(tasks)

	return
}

func (t *TaskManager) Delete(id string) (result Result[struct{}, WasmcloudTaskManagerTypesOperationError]) {
	query := "DELETE FROM tasks WHERE task_id=$1"
	params := []WasmcloudPostgres0_1_0_draft_QueryPgValue{
		WasmcloudPostgres0_1_0_draft_TypesPgValueText(id),
	}
	queryRes := WasmcloudPostgres0_1_0_draft_QueryQuery(query, params)
	if queryRes.IsErr() {
		WasiLoggingLoggingLog(WasiLoggingLoggingLevelError(), "Delete", "Got an error deleting task")
		result.SetErr(ExportsWasmcloudTaskManagerTrackerOperationError{Message: "Error deleting task"})
		return
	}

	result.Set(struct{}{})

	return
}
