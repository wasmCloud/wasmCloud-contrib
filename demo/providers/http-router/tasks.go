package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/wasmCloud/provider-sdk-go"
	wrpcnats "github.com/wrpc/wrpc/go/nats"
	"go.opentelemetry.io/otel/trace"

	tracker "github.com/wasmCloud/wasmcloud-contrib/demo/providers/http-router/bindings/wasmcloud/task_manager/tracker"
)

type Task struct {
	Id          string `json:"id"`
	Category    string `json:"category"`
	Payload     string `json:"payload"`
	CreatedAt   string `json:"created_at"`
	Done        bool   `json:"done"`
	Failure     bool   `json:"failure"`
	CompletedAt string `json:"completed_at,omitempty"`
	Result      string `json:"result,omitempty"`
}

type TaskListResponse []Task

type TasksProxy struct {
	tracer   trace.Tracer
	provider *provider.WasmcloudProvider
}

func (t *TasksProxy) createTask(w http.ResponseWriter, r *http.Request) {
	ctx, span := t.tracer.Start(r.Context(), "createTask")
	defer span.End()
	wasmAuthority := r.Host

	logger := slog.With(slog.String("authority", wasmAuthority), slog.String("method", r.Method), slog.String("path", r.URL.Path))
	logger.Info("Received task request")

	payload := "some payload"

	client := wrpcnats.NewClient(t.provider.NatsConnection(), "default.demo_image_processor-task_manager")
	resp, stop, err := tracker.Start(ctx, client, "category", payload)
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

	w.Write([]byte("Got " + *resp.Ok))
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

	w.Write([]byte(fmt.Sprintf("Got %t", resp.Ok)))
}

func (t *TasksProxy) listTasks(w http.ResponseWriter, r *http.Request) {
	ctx, span := t.tracer.Start(r.Context(), "listTasks")
	defer span.End()

	client := wrpcnats.NewClient(t.provider.NatsConnection(), "default.demo_image_processor-task_manager")
	resp, stop, err := tracker.List(ctx, client, nil)
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

	tasks := TaskListResponse{}
	for _, task := range *resp.Ok {
		respTask := Task{
			Id:        task.Id,
			Category:  task.Category,
			Payload:   task.Payload,
			CreatedAt: task.CreatedAt,
		}
		if task.Result != nil {
			respTask.Done = true
			respTask.Result = *task.Result
			respTask.CompletedAt = *task.CompletedAt
		} else if task.Failure != nil {
			respTask.Done = true
			respTask.Result = *task.Failure
			respTask.CompletedAt = *task.CompletedAt
		}
		tasks = append(tasks, respTask)
	}
	json.NewEncoder(w).Encode(tasks)
}

func (t *TasksProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// GET /api/tasks -> list
	// POST /api/tasks -> create
	// GET /api/tasks/:id -> read
	ctx, span := t.tracer.Start(r.Context(), "ServeHTTP")
	defer span.End()

	r = r.WithContext(ctx)

	if r.Method == http.MethodPost {
		t.createTask(w, r)
		return
	}

	if r.Method == http.MethodGet {
		if r.PathValue("id") != "" {
			t.getTask(w, r)
		} else {
			t.listTasks(w, r)
		}
		return
	}
}
