package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/wasmCloud/provider-sdk-go"
	wrpcnats "github.com/wrpc/wrpc/go/nats"
	"go.opentelemetry.io/otel/trace"

	tracker "github.com/wasmCloud/wasmcloud-contrib/demo/providers/http-router/bindings/wasmcloud/task_manager/tracker"
)

type Frame struct {
	Error bool        `json:"error,omitempty"`
	Data  interface{} `json:"data,omitempty"`
}

func frameData(data interface{}, error bool) Frame {
	return Frame{
		Error: error,
		Data:  data,
	}
}

type JobCreateResponse struct {
	Id string `json:"jobId"`
}

type JobResize struct {
	Done     bool   `json:"done"`
	Error    bool   `json:"error"`
	Original string `json:"original"`
	Resized  string `json:"resized"`
}

type JobAnalyze struct {
	Done  bool `json:"done"`
	Error bool `json:"error"`
	Match bool `json:"match"`
}

type Job struct {
	JobID       string     `json:"jobId"`
	Resize      JobResize  `json:"resize"`
	Analyze     JobAnalyze `json:"analyze"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt time.Time  `json:"completed_at,omitempty"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type TasksProxy struct {
	tracer   trace.Tracer
	provider *provider.WasmcloudProvider
}

func (t *TasksProxy) getTask(w http.ResponseWriter, r *http.Request) {
	ctx, span := t.tracer.Start(r.Context(), "getTask")
	defer span.End()

	taskId := r.PathValue("id")
	slog.Info("Getting task", slog.String("taskId", taskId))

	client := wrpcnats.NewClient(t.provider.NatsConnection(), "default.demo_image_processor-task_manager")
	resp, stop, err := tracker.Get(ctx, client, taskId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("error calling task handler", slog.Any("error", err))
		return
	}
	defer stop()

	if resp.Err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("error calling task handler", slog.Any("error", resp.Err))
		return
	}

	job := taskToJob(*resp.Ok)

	json.NewEncoder(w).Encode(frameData(job, false))
}

func (t *TasksProxy) listTasks(w http.ResponseWriter, r *http.Request) {
	ctx, span := t.tracer.Start(r.Context(), "listTasks")
	defer span.End()

	client := wrpcnats.NewClient(t.provider.NatsConnection(), "default.demo_image_processor-task_manager")
	resp, stop, err := tracker.List(ctx, client)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("error calling task handler", slog.Any("error", err))
		return
	}
	defer stop()

	if resp.Err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("error calling task handler", slog.Any("error", resp.Err))
		return
	}

	taskList := []Job{}
	for _, task := range *resp.Ok {
		taskList = append(taskList, taskToJob(*task))
	}

	json.NewEncoder(w).Encode(frameData(taskList, false))
}

func (t *TasksProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// GET /api/tasks -> list
	// GET /api/tasks/:id -> read
	ctx, span := t.tracer.Start(r.Context(), "ServeHTTP")
	defer span.End()

	r = r.WithContext(ctx)

	if r.Method == http.MethodGet {
		if r.PathValue("id") != "" {
			t.getTask(w, r)
		} else {
			t.listTasks(w, r)
		}
		return
	}
}

func taskToJob(trackerTask tracker.Operation) Job {
	job := Job{
		JobID:     trackerTask.Id,
		CreatedAt: time.Now(),
		Resize:    JobResize{Original: trackerTask.OriginalAsset},
	}

	if createdAt, err := time.Parse(time.RFC3339, trackerTask.CreatedAt); err == nil {
		job.CreatedAt = createdAt
	}

	if trackerTask.ResizeError != nil || trackerTask.ResizedAsset != nil {
		job.Resize.Done = true
		if trackerTask.ResizeError != nil {
			job.Resize.Error = true
		}

		if trackerTask.ResizedAsset != nil {
			job.Resize.Resized = *trackerTask.ResizedAsset
		}

		if resizeTime, err := time.Parse(time.RFC3339, *trackerTask.ResizedAt); err == nil {
			job.UpdatedAt = resizeTime
		}
	}

	if trackerTask.AnalyzeError != nil || trackerTask.AnalyzeResult != nil {
		job.Analyze.Done = true
		if trackerTask.AnalyzeError != nil {
			job.Analyze.Error = true
		}

		if trackerTask.AnalyzeResult != nil {
			job.Analyze.Match = *trackerTask.AnalyzeResult
		}

		if analyzeTime, err := time.Parse(time.RFC3339, *trackerTask.AnalyzedAt); err == nil {
			if job.UpdatedAt.IsZero() {
				job.UpdatedAt = analyzeTime
			} else {
				if analyzeTime.After(job.UpdatedAt) {
					job.CompletedAt = analyzeTime
				} else {
					job.CompletedAt = job.UpdatedAt
				}
			}
		}
	}

	return job
}
