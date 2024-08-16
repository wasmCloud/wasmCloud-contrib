package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/wasmCloud/provider-sdk-go"
	wrpcnats "github.com/wrpc/wrpc/go/nats"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/wasmCloud/wasmcloud-contrib/demo/providers/http-router/bindings/wasmcloud/image_analyzer/analyzer"
	resizer "github.com/wasmCloud/wasmcloud-contrib/demo/providers/http-router/bindings/wasmcloud/image_processor/resizer"
	tracker "github.com/wasmCloud/wasmcloud-contrib/demo/providers/http-router/bindings/wasmcloud/task_manager/tracker"
)

const MaxUploadSize = 10 * 1024 * 1024 // 10 MB
type ProcessProxy struct {
	tracer   trace.Tracer
	provider *provider.WasmcloudProvider
}

func (t *ProcessProxy) createTask(ctx context.Context, originalAsset string) (string, error) {
	client := wrpcnats.NewClient(t.provider.NatsConnection(), "default.demo_image_processor-task_manager")
	res, stop, err := tracker.Start(ctx, client, originalAsset)
	if err != nil {
		return "", err
	}
	defer stop()

	if res.Err != nil {
		return "", fmt.Errorf("error creating task: %s", res.Err.Message)
	}

	return *res.Ok, nil
}

func (t *ProcessProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := t.tracer.Start(r.Context(), "ServeHTTP")
	defer span.End()

	if err := r.ParseMultipartForm(MaxUploadSize); err != nil {
		http.Error(w, "The uploaded file is too big. Please choose an file that's less than 10MB in size", http.StatusBadRequest)
		return
	}

	image, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer image.Close()

	imageBytes, err := io.ReadAll(image)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	client := wrpcnats.NewClient(t.provider.NatsConnection(), "default.demo_image_processor-image_processor")

	res, stop, err := resizer.Upload(ctx, client, imageBytes)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("error calling resizer", slog.Any("error", err))
		return
	}
	defer stop()

	if res.Err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("error calling resizer", slog.String("error", *res.Err))
		return
	}

	originalAsset := *res.Ok
	taskId, err := t.createTask(ctx, originalAsset)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("error calling resizer", slog.Any("error", err))
		return
	}

	resp := frameData(JobCreateResponse{Id: taskId}, false)
	json.NewEncoder(w).Encode(resp)

	span.SetStatus(codes.Ok, "Served Request")

	go t.resizeTask(taskId, originalAsset)
	go t.analyzeTask(taskId, imageBytes)
}

func (t *ProcessProxy) resizeTask(taskId string, originalAsset string) {
	ctx := context.Background()

	taskClient := wrpcnats.NewClient(t.provider.NatsConnection(), "default.demo_image_processor-task_manager")
	client := wrpcnats.NewClient(t.provider.NatsConnection(), "default.demo_image_processor-image_processor")

	res, stop, err := resizer.Resize(ctx, client, originalAsset, 100, 100)
	if err != nil {
		slog.Error("error calling resizer", slog.Any("error", err))
		errStr := err.Error()
		_, completeStop, err := tracker.CompleteResize(ctx, taskClient, taskId, nil, &errStr)
		if err != nil {
			slog.Error("error calling tracker", slog.Any("error", err))
			return
		}
		completeStop()
		return
	}
	defer stop()

	_, completeStop, err := tracker.CompleteResize(ctx, taskClient, taskId, res.Ok, nil)
	if err != nil {
		slog.Error("error calling tracker", slog.Any("error", err))
		return
	}
	completeStop()
}

func (t *ProcessProxy) analyzeTask(taskId string, imageBytes []byte) {
	ctx := context.Background()

	taskClient := wrpcnats.NewClient(t.provider.NatsConnection(), "default.demo_image_processor-task_manager")
	client := wrpcnats.NewClient(t.provider.NatsConnection(), "default.demo_image_processor-image_analyzer")

	res, stop, err := analyzer.Detect(ctx, client, imageBytes)
	if err != nil {
		slog.Error("error calling analyzer", slog.Any("error", err))
		errStr := err.Error()
		_, completeStop, err := tracker.CompleteAnalyze(ctx, taskClient, taskId, nil, &errStr)
		if err != nil {
			slog.Error("error calling tracker", slog.Any("error", err))
			return
		}
		completeStop()
		return
	}
	defer stop()

	_, completeStop, err := tracker.CompleteAnalyze(ctx, taskClient, taskId, res.Ok, nil)
	if err != nil {
		slog.Error("error calling tracker", slog.Any("error", err))
		return
	}
	completeStop()
}
