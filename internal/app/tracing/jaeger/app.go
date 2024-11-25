package jaegerapp

import (
	"context"
	"log/slog"

	"github.com/BariVakhidov/sso/internal/lib/logger/sl"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

type App struct {
	log           *slog.Logger
	traceProvider *trace.TracerProvider
}

func New(log *slog.Logger, jaegerEndpoint string) (*App, error) {
	const op = "tracing.jaeger.New"
	// Set up an OTLP exporter (HTTP)
	exporter, err := otlptracehttp.New(
		context.Background(),
		otlptracehttp.WithEndpoint(jaegerEndpoint),
		otlptracehttp.WithInsecure(),
	)

	if err != nil {
		log.Error("failed to create otlptracehttp", slog.String("op", op), sl.Err(err))
		return nil, err
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("sso"),
		)),
	)
	otel.SetTracerProvider(tp)

	return &App{
		traceProvider: tp,
		log:           log,
	}, nil
}

func (a *App) Stop(ctx context.Context) {
	const op = "tracing.jaeger.Stop"
	log := a.log.With(slog.String("op", op))
	log.Info("stopping jaeger tracer")

	if err := a.traceProvider.Shutdown(ctx); err != nil {
		log.Error("failed to stop jaeger tracer", sl.Err(err))
	}
}
