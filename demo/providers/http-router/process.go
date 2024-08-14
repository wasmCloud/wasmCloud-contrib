package main

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/wasmCloud/provider-sdk-go"
	wrpcnats "github.com/wrpc/wrpc/go/nats"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	resizer "github.com/wasmCloud/wasmcloud-contrib/demo/providers/http-router/bindings/wasmcloud/image_processor/resizer"
)

const MaxUploadSize = 10 * 1024 * 1024 // 10 MB
type ProcessProxy struct {
	tracer   trace.Tracer
	provider *provider.WasmcloudProvider
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

	res, stop, err := resizer.Resize(ctx, client, imageBytes, 100, 100)
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

	span.SetStatus(codes.Ok, "Served Request")
}
