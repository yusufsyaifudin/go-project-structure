package oteltracer

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

type ExporterOpt func(*ExporterOption) error

// WithLogger set logger instance for the STDOUT span exporter.
func WithLogger(logger io.Writer) ExporterOpt {
	return func(option *ExporterOption) error {
		if logger == nil {
			return fmt.Errorf("cannot use nil logger")
		}

		option.logger = logger
		return nil
	}
}

// WithJaegerEndpoint set Jaeger endpoint for type JAEGER span exporter.
// Default to http://localhost:14268/api/traces
func WithJaegerEndpoint(endpoint string) ExporterOpt {
	return func(option *ExporterOption) error {
		option.jaegerEndpoint = endpoint
		return nil
	}
}

// WithOTLPEndpoint set OpenTelemetry endpoint collector for type OTLP span exporter.
// Default to localhost:4318
func WithOTLPEndpoint(endpoint string) ExporterOpt {
	return func(option *ExporterOption) error {
		option.otlpEndpoint = endpoint
		return nil
	}
}

type ExporterOption struct {
	logger         io.Writer
	jaegerEndpoint string
	otlpEndpoint   string
}

// NewTracerExporter select the tracer span exporter based on name.
// Default to noop exporter if no name or NOOP specified.
func NewTracerExporter(name string, opts ...ExporterOpt) (trace.SpanExporter, error) {
	cfg := &ExporterOption{
		logger:         os.Stdout,
		jaegerEndpoint: "http://localhost:14268/api/traces",
		otlpEndpoint:   "localhost:4318",
	}

	for _, opt := range opts {
		err := opt(cfg)
		if err != nil {
			return nil, err
		}
	}

	name = strings.TrimSpace(name)
	name = strings.ToUpper(name)
	switch name {
	case "JAEGER":
		endpoint := strings.TrimSpace(cfg.jaegerEndpoint)
		if endpoint == "" {
			return nil, fmt.Errorf("cannot use OpenTelemetry JAEGER if OTEL_EXPORTER_JAEGER_ENDPOINT is empty")
		}

		return jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(endpoint)))

	case "OTLP":
		endpoint := strings.TrimSpace(cfg.otlpEndpoint)
		if endpoint == "" {
			return nil, fmt.Errorf("cannot use OpenTelemetry OTLP if OTEL_EXPORTER_OTLP_ENDPOINT is empty")
		}

		return otlptrace.New(
			context.Background(),
			otlptracehttp.NewClient(
				otlptracehttp.WithInsecure(),
				otlptracehttp.WithEndpoint(endpoint),
			),
		)

	case "STDOUT":
		return stdouttrace.New(
			stdouttrace.WithWriter(cfg.logger),
			// Use human-readable output.
			stdouttrace.WithPrettyPrint(),
			// Do not print timestamps for the demo.
			stdouttrace.WithoutTimestamps(),
		)

	case "", "NOOP":
		return tracetest.NewNoopExporter(), nil
	default:
		return nil, fmt.Errorf("unknown name='%s' for OpenTelemetry span exporter", name)
	}
}
