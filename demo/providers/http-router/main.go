//go:generate wit-bindgen-wrpc go --out-dir bindings --package github.com/wasmCloud/wasmcloud-contrib/demo/providers/http-router/bindings wit

package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	//	_ "net/http/pprof"

	"github.com/gorilla/handlers"
	"github.com/wasmCloud/provider-sdk-go"
	"go.opentelemetry.io/otel"
)

//go:embed static
var staticAssets embed.FS

func run() error {
	server := &Server{
		tracer: otel.Tracer("http-router"),
	}

	p, err := provider.New(
		provider.HealthCheck(func() string {
			return server.HealthCheck()
		}),
		provider.Shutdown(func() error {
			return server.Shutdown()
		}),
	)
	if err != nil {
		return err
	}
	server.provider = p

	var port int
	if rawServerPort, ok := p.HostData().Config["port"]; !ok {
		slog.Error("Port not specified in 'provider_config'")
		os.Exit(1)
	} else {
		if port, err = strconv.Atoi(rawServerPort); err != nil {
			slog.Error("Couldn't parse desired port number", slog.String("requested_port", rawServerPort))
			os.Exit(1)
		}
	}

	assets, err := fs.Sub(staticAssets, "static")
	if err != nil {
		slog.Error("Couldn't find static assets", slog.Any("error", err))
		os.Exit(1)
	}

	mux := http.NewServeMux()
	tasksProxy := &TasksProxy{provider: p, tracer: otel.Tracer("tasks-proxy")}
	analyzeProxy := &AnalyzeProxy{provider: p, tracer: otel.Tracer("analyze-proxy")}
	processProxy := &ProcessProxy{provider: p, tracer: otel.Tracer("process-proxy")}
	statisticsProxy := &StatisticsProxy{provider: p, tracer: otel.Tracer("statistics-proxy")}
	serveProxy := &ServeProxy{provider: p, tracer: otel.Tracer("statistics-proxy")}
	mux.Handle("/api/tasks", tasksProxy)
	mux.Handle("/api/tasks/{id...}", tasksProxy)
	mux.Handle("/api/statistics", statisticsProxy)
	mux.Handle("/api/process", processProxy)
	mux.Handle("/api/analyze", analyzeProxy)
	mux.Handle("/blob/{asset...}", serveProxy)

	fs := http.FileServer(http.FS(assets))
	mux.Handle("/", http.StripPrefix("/", fs))

	httpHandler := http.Handler(mux)
	// Reads X-Forwarded-* headers
	if serverMode, ok := p.HostData().Config["mode"]; ok {
		if serverMode == "behind_proxy" {
			httpHandler = handlers.ProxyHeaders(httpHandler)
		}
	}

	server.httpServer = &http.Server{
		Handler: httpHandler,
		Addr:    fmt.Sprintf(":%d", port),
	}

	go func() {
		server.httpServer.ListenAndServe()
	}()

	providerCh := make(chan error, 1)
	signalCh := make(chan os.Signal, 1)

	go func() {
		err := p.Start()
		providerCh <- err
	}()

	signal.Notify(signalCh, syscall.SIGINT)

	select {
	case err = <-providerCh:
		return err
	case <-signalCh:
		p.Shutdown()
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		slog.Error("Couldn't run server", slog.Any("error", err))
		os.Exit(1)
	}
}
