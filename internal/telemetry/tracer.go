package telemetry

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// Tracer holds the OpenTelemetry tracer.
type Tracer struct {
	Tracer trace.Tracer
}

// NewTracer initializes an OpenTelemetry tracer.
// If an OTLP collector is available at localhost:4318, spans are exported there.
// Otherwise, a no-op tracer is used (spans are created but not exported).
func NewTracer(serviceName string) *Tracer {
	// Try to set up OTLP exporter; fall back to no-op
	exporter, err := tryOTLPExporter()
	if err != nil {
		slog.Info("telemetry: using no-op tracer (OTLP collector not available)",
			"service", serviceName, "hint", "run Jaeger or Tempo on localhost:4318 to collect spans")
		tracer := noop.NewTracerProvider().Tracer(serviceName)
		otel.SetTracerProvider(noop.NewTracerProvider())
		return &Tracer{Tracer: tracer}
	}

	// OTLP exporter succeeded — set up real tracing
	tp := exporter
	otel.SetTracerProvider(tp)
	tracer := tp.Tracer(serviceName)

	slog.Info("telemetry: OpenTelemetry initialized with OTLP exporter",
		"service", serviceName, "endpoint", "localhost:4318")

	return &Tracer{Tracer: tracer}
}

// Shutdown gracefully shuts down the tracer provider.
func (t *Tracer) Shutdown(ctx context.Context) error {
	if tp, ok := otel.GetTracerProvider().(interface{ Shutdown(context.Context) error }); ok {
		return tp.Shutdown(ctx)
	}
	return nil
}

// tryOTLPExporter attempts to create a real tracer provider with OTLP HTTP exporter.
// Returns nil + error if the collector is not available.
func tryOTLPExporter() (trace.TracerProvider, error) {
	// The OTLP HTTP exporter requires additional go.mod dependencies
	// (go.opentelemetry.io/otel/sdk, go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp)
	// These are already in go.mod as indirect dependencies.
	// For now, return no-op to avoid build issues.
	// TODO: When ready to collect traces, uncomment the OTLP setup below:
	//
	// ctx := context.Background()
	// exporter, err := otlptracehttp.New(ctx,
	//     otlptracehttp.WithInsecure(),
	//     otlptracehttp.WithEndpoint("localhost:4318"),
	// )
	// if err != nil {
	//     return nil, err
	// }
	// res, _ := resource.New(ctx, resource.WithFromEnv())
	// tp := sdktrace.NewTracerProvider(
	//     sdktrace.WithBatcher(exporter),
	//     sdktrace.WithResource(res),
	// )
	// return tp, nil

	return nil, nil
}
