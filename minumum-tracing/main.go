package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"time"

	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type Response struct {
	Message string `json:"message"`
}

var (
	echoPathRegex = regexp.MustCompile(`^/echo/(.+)$`)
)

type server struct {
	tracer trace.Tracer
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/_ah/health" && r.Method == http.MethodGet:
		s.handleHealth(w, r)
	case echoPathRegex.MatchString(r.URL.Path) && r.Method == http.MethodGet:
		s.handleEcho(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (s *server) handleEcho(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx, span := s.tracer.Start(ctx, "echo-handler")
	defer span.End()

	matches := echoPathRegex.FindStringSubmatch(r.URL.Path)
	if len(matches) != 2 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	message := matches[1]

	slog.InfoContext(ctx, "received echo request",
		"message", message,
		"trace_id", span.SpanContext().TraceID().String(),
	)

	time.Sleep(100 * time.Millisecond)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Message: message})
}

func main() {
	ctx := context.Background()

	exp, err := autoexport.NewSpanExporter(ctx)
	if err != nil {
		slog.Error("failed to create span exporter", "error", err)
		os.Exit(1)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	defer tp.ForceFlush(ctx)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	srv := &server{
		tracer: otel.Tracer(""),
	}

	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: srv,
	}

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt)
		<-sigChan

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		slog.Info("shutting down server")
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("server shutdown failed", "error", err)
		}
	}()

	slog.Info("starting server", "addr", httpServer.Addr)
	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
