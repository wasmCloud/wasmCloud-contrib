package main

import (
	"log/slog"
	"net/http"

	"github.com/wasmCloud/provider-sdk-go"
	"go.opentelemetry.io/otel/trace"
)

type StatisticsProxy struct {
	tracer   trace.Tracer
	provider *provider.WasmcloudProvider
}

func (t *StatisticsProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := t.tracer.Start(r.Context(), "ServeHTTP")
	defer span.End()

	wasmAuthority := r.Host

	logger := slog.With(slog.String("authority", wasmAuthority), slog.String("method", r.Method), slog.String("path", r.URL.Path))
	logger.Info("Received request")
	_ = ctx
}
