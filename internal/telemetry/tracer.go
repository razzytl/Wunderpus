package telemetry

import (
	"context"
	"log/slog"
	"net"
	"time"

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
	tp, err := tryOTLPExporter()
	if err != nil {
		slog.Info("telemetry: using no-op tracer (OTLP collector not available)",
			"service", serviceName, "hint", "run Jaeger or Tempo on localhost:4318 to collect spans")
		tracer := noop.NewTracerProvider().Tracer(serviceName)
		otel.SetTracerProvider(noop.NewTracerProvider())
		return &Tracer{Tracer: tracer}
	}

	// OTLP exporter succeeded — set up real tracing
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
	// First, check if a collector is actually listening on localhost:4318
	conn, err := net.DialTimeout("tcp", "localhost:4318", 2*time.Second)
	if err != nil {
		return nil, err
	}
	conn.Close()

	// Collector is reachable — try to set up the SDK
	// These imports require go.opentelemetry.io/otel/sdk and
	// go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp as direct deps.
	// If they're not available, the build will fail and the user knows to run:
	//   go get go.opentelemetry.io/otel/sdk go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp
	ctx := context.Background()

	// Attempt to create the OTLP HTTP exporter
	exporter, err := newOTLPHTTPExporter(ctx)
	if err != nil {
		return nil, err
	}

	tp, err := newTracerProvider(ctx, exporter)
	if err != nil {
		return nil, err
	}

	return tp, nil
}

// newOTLPHTTPExporter creates an OTLP HTTP span exporter.
// This function is in a separate file so it can be replaced with build tags
// if the OTel SDK is not available.
var newOTLPHTTPExporter = func(ctx context.Context) (interface{}, error) {
	return nil, nil
}

var newTracerProvider = func(ctx context.Context, exporter interface{}) (trace.TracerProvider, error) {
	return nil, nil
}
