package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	httpserver "github.com/PycMono/go-context-sdk/example/http"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGHUP,
		syscall.SIGQUIT,
		syscall.SIGTERM,
		syscall.SIGINT,
	)
	defer stop()

	listener, err := net.Listen("tcp", ":5005")
	if err != nil {
		panic(err)
	}
	if err := run(ctx, listener); err != nil {
		panic(err)
	}
}

func run(ctx context.Context, listener net.Listener) error {
	tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.NeverSample()))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	handler, err := httpserver.NewServer("", "", "")
	if err != nil {
		return errors.Join(err, tp.Shutdown(context.Background()))
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/span", handler.SpanHandler)
	mux.HandleFunc("/test-span", handler.Redirect)
	server := &http.Server{Handler: mux}
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(listener) }()

	var serveResult, shutdownResult error
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		shutdownResult = server.Shutdown(shutdownCtx)
		cancel()
		serveResult = <-serveErr
	case serveResult = <-serveErr:
	}
	if errors.Is(serveResult, http.ErrServerClosed) {
		serveResult = nil
	}
	return errors.Join(serveResult, shutdownResult, tp.Shutdown(context.Background()))
}
