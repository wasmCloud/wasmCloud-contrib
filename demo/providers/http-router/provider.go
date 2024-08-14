package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wasmCloud/provider-sdk-go"
	wasitypes "github.com/wasmCloud/wasmcloud-contrib/demo/providers/http-router/bindings/wasi/http/types"
	wrpc "github.com/wrpc/wrpc/go"
	"go.opentelemetry.io/otel/trace"
)

type wasiBodyTrailerSplitter struct {
	bodyRx         io.Reader
	trailer        http.Header
	trailerRx      wrpc.Receiver[[]*wrpc.Tuple2[string, [][]byte]]
	trailerOnce    sync.Once
	trailerIsReady uint32
}

func (r *wasiBodyTrailerSplitter) Read(b []byte) (int, error) {
	n, err := r.bodyRx.Read(b)
	if err != io.EOF {
		return n, err
	}

	r.trailerOnce.Do(func() {
		trailers, err := r.trailerRx.Receive()
		if err != nil {
			slog.Debug("wasmBodyReader: Failed to read trailers")
			return
		}
		for _, header := range trailers {
			for _, value := range header.V1 {
				r.trailer.Add(header.V0, string(value))
			}
		}
		atomic.CompareAndSwapUint32(&r.trailerIsReady, 0, 1)
	})

	return n, io.EOF
}

func (r *wasiBodyTrailerSplitter) TrailerIsReady() bool {
	return atomic.LoadUint32(&r.trailerIsReady) == 1
}

type trailerReader struct {
	trailer    http.Header
	bodyIsDone uint32
}

func (t *trailerReader) wake() {
	atomic.CompareAndSwapUint32(&t.bodyIsDone, 0, 1)
}

func (t *trailerReader) IsComplete() bool {
	slog.Info("trailerReader.IsComplete", "done", atomic.LoadUint32(&t.bodyIsDone))
	return atomic.LoadUint32(&t.bodyIsDone) == 1
}

func (t *trailerReader) Receive() ([]*wrpc.Tuple2[string, [][]byte], error) {
	if !t.IsComplete() {
		slog.Info("trailerReader.Receive short buffer")
		return nil, io.ErrShortBuffer
	}

	slog.Info("trailerReader.Receive", slog.Any("trailer", t.trailer))
	return httpHeaderToWasi(t.trailer), nil
}

type bodyReader struct {
	body          io.ReadCloser
	trailerReader *trailerReader
}

func (r *bodyReader) Read(b []byte) (int, error) {
	return r.body.Read(b)
}

func (r *bodyReader) Close() error {
	slog.Info("Closing bodyReader")
	r.trailerReader.wake()
	// not needed as http.Server will close the body
	return nil
}

func (r *bodyReader) IsComplete() bool {
	slog.Info("bodyReader IsComplete")
	return true
}

func httpBodyTrailerSplitter(body io.ReadCloser, trailer http.Header) (wrpc.ReadCompleter, wrpc.ReceiveCompleter[[]*wrpc.Tuple2[string, [][]uint8]]) {
	trailerReader := &trailerReader{trailer: trailer}
	bodyReader := &bodyReader{body: body, trailerReader: trailerReader}
	return bodyReader, trailerReader
}

type Server struct {
	provider   *provider.WasmcloudProvider
	httpServer *http.Server
	tracer     trace.Tracer
}

func methodToWasi(method string) *wasitypes.Method {
	switch method {
	case http.MethodConnect:
		return wasitypes.NewMethodConnect()
	case http.MethodGet:
		return wasitypes.NewMethodGet()
	case http.MethodHead:
		return wasitypes.NewMethodHead()
	case http.MethodPost:
		return wasitypes.NewMethodPost()
	case http.MethodPut:
		return wasitypes.NewMethodPut()
	case http.MethodPatch:
		return wasitypes.NewMethodPatch()
	case http.MethodDelete:
		return wasitypes.NewMethodDelete()
	case http.MethodOptions:
		return wasitypes.NewMethodOptions()
	case http.MethodTrace:
		return wasitypes.NewMethodTrace()
	default:
		return wasitypes.NewMethodOther(method)
	}
}

func schemeToWasi(scheme string) *wasitypes.Scheme {
	switch scheme {
	case "http":
		return wasitypes.NewSchemeHttp()
	case "https":
		return wasitypes.NewSchemeHttps()
	default:
		return wasitypes.NewSchemeOther(scheme)
	}
}

func httpHeaderToWasi(header http.Header) []*wrpc.Tuple2[string, [][]uint8] {
	wasiHeader := make([]*wrpc.Tuple2[string, [][]uint8], 0, len(header))
	for k, vals := range header {
		var uintVals [][]uint8
		for _, v := range vals {
			uintVals = append(uintVals, []byte(v))
		}
		wasiHeader = append(wasiHeader, &wrpc.Tuple2[string, [][]uint8]{
			V0: k,
			V1: uintVals,
		})
	}

	return wasiHeader
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := s.tracer.Start(r.Context(), "ServeHTTP")
	defer span.End()

	wasmAuthority := r.Host

	logger := slog.With(slog.String("authority", wasmAuthority), slog.String("method", r.Method), slog.String("path", r.URL.Path))
	logger.Info("Received request")
	_ = ctx

	// client := wrpcnats.NewClient(s.provider.NatsConnection(), targetLink.target)
	//
	// bodyReader, trailerReader := httpBodyTrailerSplitter(r.Body, r.Trailer)
	// req := &wrpctypes.Request{
	// 	Headers:       httpHeaderToWasi(r.Header),
	// 	Method:        methodToWasi(r.Method),
	// 	Scheme:        schemeToWasi(r.URL.Scheme),
	// 	PathWithQuery: &r.RequestURI,
	// 	Authority:     &wasmAuthority,
	// 	Trailers:      trailerReader,
	// 	Body:          bodyReader,
	// }
	//
	// reqCtx, reqCancel := context.WithTimeout(ctx, 1*time.Minute)
	// defer reqCancel()
	//
	// resp, stop, err := incoming_handler.Handle(reqCtx, client, req)
	// if err != nil {
	// 	w.WriteHeader(http.StatusInternalServerError)
	// 	logger.Error("Sending request to wasmbus", slog.Any("error", err))
	// 	span.SetStatus(codes.Error, "Failed to send request to wasmbus")
	// 	return
	// }
	// defer stop()
	//
	// for _, hdr := range resp.Ok.Headers {
	// 	for _, hdrVal := range hdr.V1 {
	// 		w.Header().Add(hdr.V0, string(hdrVal))
	// 	}
	// }
	//
	// w.WriteHeader(int(resp.Ok.Status))
	//
	// wasiBodyTrailer := &wasiBodyTrailerSplitter{bodyRx: resp.Ok.Body, trailerRx: resp.Ok.Trailers}
	// _, err = io.Copy(w, wasiBodyTrailer)
	// if err != nil {
	// 	logger.Error("Couldn't write body", slog.Any("error", err))
	// 	span.SetStatus(codes.Error, "Couldn't write body")
	// 	return
	// }
	//
	// if wasiBodyTrailer.TrailerIsReady() {
	// 	for k, vals := range wasiBodyTrailer.trailer {
	// 		for _, v := range vals {
	// 			w.Header().Add(k, v)
	// 		}
	// 	}
	// }
	//
	// endTime := time.Now()
	// deltaTime := endTime.Sub(startTime)
	// _ = deltaTime
	// //	logger.Info("Served request", slog.Any("status", resp.Ok.Status), slog.Duration("duration", deltaTime))
	// span.SetStatus(codes.Ok, "Served Request")
}

func (s *Server) HealthCheck() string {
	return "healthy"
}

func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}
