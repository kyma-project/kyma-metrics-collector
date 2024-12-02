package otel

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/trace"
)

// SetupSDK bootstraps the OpenTelemetry pipeline.
// If it does not return an error, make sure to call shutdown for proper cleanup.
func SetupSDK(ctx context.Context) (func(context.Context) error, error) {
	tracerProvider, err := newTraceProvider()
	if err != nil {
		return nil, err
	}

	otel.SetTracerProvider(tracerProvider)

	return tracerProvider.Shutdown, nil
}

func newTraceProvider() (*trace.TracerProvider, error) {
	traceExporter, err := otlptrace.New(
		context.Background(),
		otlptracegrpc.NewClient(),
	)
	if err != nil {
		return nil, err
	}

	traceProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter),
	)

	return traceProvider, nil
}
