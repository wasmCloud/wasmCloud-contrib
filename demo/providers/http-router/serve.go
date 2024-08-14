package main

import (
	"log/slog"
	"net/http"

	"github.com/wasmCloud/provider-sdk-go"
	wrpcnats "github.com/wrpc/wrpc/go/nats"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	resizer "github.com/wasmCloud/wasmcloud-contrib/demo/providers/http-router/bindings/wasmcloud/image_processor/resizer"
)

type ServeProxy struct {
	tracer   trace.Tracer
	provider *provider.WasmcloudProvider
}

func (t *ServeProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := t.tracer.Start(r.Context(), "ServeHTTP")
	defer span.End()

	assetKey := r.PathValue("asset")

	if assetKey == "" {
		http.Error(w, "Missing asset key", http.StatusBadRequest)
		return
	}

	client := wrpcnats.NewClient(t.provider.NatsConnection(), "default.demo_image_processor-image_processor")

	res, stop, err := resizer.Serve(ctx, client, assetKey)
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

	w.Write(*res.Ok)

	span.SetStatus(codes.Ok, "Served Request")
}
