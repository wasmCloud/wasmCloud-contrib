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

func (t *TaskManager) Start(originalAsset string) (result Result[string, ExportsWasmcloudTaskManagerTrackerOperationError]) {
	query := "INSERT INTO tasks(original_asset) VALUES($1) RETURNING task_id"
	params := []WasmcloudPostgres0_1_0_draft_QueryPgValue{
		WasmcloudPostgres0_1_0_draft_TypesPgValueText(originalAsset),
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

func (t *TaskManager) CompleteAnalyze(id string, detected Option[bool], analyzeError Option[string]) (result Result[struct{}, ExportsWasmcloudTaskManagerTrackerOperationError]) {
	if analyzeError.IsSome() {
		query := fmt.Sprintf("UPDATE tasks set analyzed_at=NOW(), analyze_error=$1 WHERE task_id = '%s'", id)
		params := []WasmcloudPostgres0_1_0_draft_QueryPgValue{
			WasmcloudPostgres0_1_0_draft_TypesPgValueText(analyzeError.Unwrap()),
		}
		queryRes := WasmcloudPostgres0_1_0_draft_QueryQuery(query, params)
		if queryRes.IsErr() {
			WasiLoggingLoggingLog(WasiLoggingLoggingLevelError(), "CompleteAnalyze", "Got an error from query")
			result.SetErr(ExportsWasmcloudTaskManagerTrackerOperationError{Message: "Error updating task"})
			return
		}

		result.Set(struct{}{})
		return
	}

	if detected.IsNone() {
		WasiLoggingLoggingLog(WasiLoggingLoggingLevelError(), "CompleteAnalyze", "Must provide either detection or error")
		result.SetErr(ExportsWasmcloudTaskManagerTrackerOperationError{Message: "Must provide either detection or error"})
		return
	}

	query := fmt.Sprintf("UPDATE tasks set analyzed_at=NOW(), analyze_result=$1 WHERE task_id = '%s'", id)
	params := []WasmcloudPostgres0_1_0_draft_QueryPgValue{
		WasmcloudPostgres0_1_0_draft_TypesPgValueBool(detected.Unwrap()),
	}
	queryRes := WasmcloudPostgres0_1_0_draft_QueryQuery(query, params)
	if queryRes.IsErr() {
		WasiLoggingLoggingLog(WasiLoggingLoggingLevelError(), "CompleteAnalyze", "Got an error from query")
		result.SetErr(ExportsWasmcloudTaskManagerTrackerOperationError{Message: "Error updating task"})
		return
	}

	result.Set(struct{}{})

	return
}

func (t *TaskManager) CompleteResize(id string, resizedAsset Option[string], resizedError Option[string]) (result Result[struct{}, ExportsWasmcloudTaskManagerTrackerOperationError]) {
	if resizedError.IsSome() {
		query := fmt.Sprintf("UPDATE tasks set resized_at=NOW(), resize_error=$1 WHERE task_id = '%s'", id)
		params := []WasmcloudPostgres0_1_0_draft_QueryPgValue{
			WasmcloudPostgres0_1_0_draft_TypesPgValueText(resizedError.Unwrap()),
		}
		queryRes := WasmcloudPostgres0_1_0_draft_QueryQuery(query, params)
		if queryRes.IsErr() {
			WasiLoggingLoggingLog(WasiLoggingLoggingLevelError(), "CompleteResize", "Got an error from query")
			result.SetErr(ExportsWasmcloudTaskManagerTrackerOperationError{Message: "Error updating task"})
			return
		}

		result.Set(struct{}{})
		return
	}

	if resizedAsset.IsNone() {
		WasiLoggingLoggingLog(WasiLoggingLoggingLevelError(), "CompleteResize", "Must provide either resized asset or error")
		result.SetErr(ExportsWasmcloudTaskManagerTrackerOperationError{Message: "Must provide either resized asset or error"})
		return
	}

	query := fmt.Sprintf("UPDATE tasks set resized_at=NOW(), resized_asset=$1 WHERE task_id = '%s'", id)
	params := []WasmcloudPostgres0_1_0_draft_QueryPgValue{
		WasmcloudPostgres0_1_0_draft_TypesPgValueText(resizedAsset.Unwrap()),
	}
	queryRes := WasmcloudPostgres0_1_0_draft_QueryQuery(query, params)
	if queryRes.IsErr() {
		WasiLoggingLoggingLog(WasiLoggingLoggingLevelError(), "CompleteResize", "Got an error from query")
		result.SetErr(ExportsWasmcloudTaskManagerTrackerOperationError{Message: "Error updating task"})
		return
	}

	result.Set(struct{}{})

	return
}

func (t *TaskManager) Get(id string) (result Result[ExportsWasmcloudTaskManagerTrackerOperation, ExportsWasmcloudTaskManagerTrackerOperationError]) {
	query := `SELECT 
		task_id,original_asset,created_at,
		resize_error,resized_asset,resized_at,
		analyze_error,analyze_result,analyzed_at
	 FROM tasks WHERE task_id=$1 LIMIT 1`
	params := []WasmcloudPostgres0_1_0_draft_QueryPgValue{
		WasmcloudPostgres0_1_0_draft_TypesPgValueText(id),
	}
	queryRes := WasmcloudPostgres0_1_0_draft_QueryQuery(query, params)
	if queryRes.IsErr() {
		WasiLoggingLoggingLog(WasiLoggingLoggingLevelError(), "List", "Got an error from query")
		result.SetErr(ExportsWasmcloudTaskManagerTrackerOperationError{Message: "Error listing tasks"})
		return
	}

	rows := queryRes.Unwrap()
	if len(rows) == 0 {
		result.SetErr(ExportsWasmcloudTaskManagerTrackerOperationError{Message: "Task not found"})
		return
	}
	row := rows[0]
	resizedAsset := Option[string]{}
	resizeError := Option[string]{}
	resizedAt := Option[string]{}

	analyzeResult := Option[bool]{}
	analyzeError := Option[string]{}
	analyzedAt := Option[string]{}

	if row[3].Value.Kind() != WasmcloudPostgres0_1_0_draft_TypesPgValueKindNull {
		resizeError.Set(row[3].Value.GetText())
	}
	if row[4].Value.Kind() != WasmcloudPostgres0_1_0_draft_TypesPgValueKindNull {
		resizedAsset.Set(row[4].Value.GetText())
	}
	if row[5].Value.Kind() != WasmcloudPostgres0_1_0_draft_TypesPgValueKindNull {
		resizedAt.Set(formatTimestamp(row[5].Value.GetTimestamp()))
	}

	if row[6].Value.Kind() != WasmcloudPostgres0_1_0_draft_TypesPgValueKindNull {
		analyzeError.Set(row[6].Value.GetText())
	}
	if row[7].Value.Kind() != WasmcloudPostgres0_1_0_draft_TypesPgValueKindNull {
		analyzeResult.Set(row[7].Value.GetBool())
	}
	if row[8].Value.Kind() != WasmcloudPostgres0_1_0_draft_TypesPgValueKindNull {
		analyzedAt.Set(formatTimestamp(row[8].Value.GetTimestamp()))
	}

	result.Set(WasmcloudTaskManagerTypesOperation{
		Id:            row[0].Value.GetText(),
		OriginalAsset: row[1].Value.GetText(),
		CreatedAt:     formatTimestamp(row[2].Value.GetTimestamp()),

		ResizedAsset: resizedAsset,
		ResizeError:  resizeError,
		ResizedAt:    resizedAt,

		AnalyzeResult: analyzeResult,
		AnalyzeError:  analyzeError,
		AnalyzedAt:    analyzedAt,
	})

	return
}

func (t *TaskManager) List() (result Result[[]ExportsWasmcloudTaskManagerTrackerOperation, ExportsWasmcloudTaskManagerTrackerOperationError]) {
	query := `SELECT 
		task_id,original_asset,created_at,
		resize_error,resized_asset,resized_at,
		analyze_error,analyze_result,analyzed_at
	 FROM tasks`
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
		resizedAsset := Option[string]{}
		resizeError := Option[string]{}
		resizedAt := Option[string]{}

		analyzeResult := Option[bool]{}
		analyzeError := Option[string]{}
		analyzedAt := Option[string]{}

		if row[3].Value.Kind() != WasmcloudPostgres0_1_0_draft_TypesPgValueKindNull {
			resizeError.Set(row[3].Value.GetText())
		}
		if row[4].Value.Kind() != WasmcloudPostgres0_1_0_draft_TypesPgValueKindNull {
			resizedAsset.Set(row[4].Value.GetText())
		}
		if row[5].Value.Kind() != WasmcloudPostgres0_1_0_draft_TypesPgValueKindNull {
			resizedAt.Set(formatTimestamp(row[5].Value.GetTimestamp()))
		}

		if row[6].Value.Kind() != WasmcloudPostgres0_1_0_draft_TypesPgValueKindNull {
			analyzeError.Set(row[6].Value.GetText())
		}
		if row[7].Value.Kind() != WasmcloudPostgres0_1_0_draft_TypesPgValueKindNull {
			analyzeResult.Set(row[7].Value.GetBool())
		}
		if row[8].Value.Kind() != WasmcloudPostgres0_1_0_draft_TypesPgValueKindNull {
			analyzedAt.Set(formatTimestamp(row[8].Value.GetTimestamp()))
		}

		tasks = append(tasks, WasmcloudTaskManagerTypesOperation{
			Id:            row[0].Value.GetText(),
			OriginalAsset: row[1].Value.GetText(),
			CreatedAt:     formatTimestamp(row[2].Value.GetTimestamp()),

			ResizedAsset: resizedAsset,
			ResizeError:  resizeError,
			ResizedAt:    resizedAt,

			AnalyzeResult: analyzeResult,
			AnalyzeError:  analyzeError,
			AnalyzedAt:    analyzedAt,
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

func formatTimestamp(t WasmcloudPostgres0_1_0_draft_TypesTimestamp) string {
	date := t.Date.GetYmd()
	return fmt.Sprintf("%02d-%02d-%02dT%02d:%02d:%02dZ", date.F0, date.F1, date.F2, t.Time.Hour, t.Time.Min, t.Time.Sec)
}
