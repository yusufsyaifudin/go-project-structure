package oteltracer

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

type ExporterOpt func(*ExporterOption) error

// WithLogger set logger instance for the STDOUT span exporter.
func WithLogger(logger *slog.Logger) ExporterOpt {
	return func(option *ExporterOption) error {
		if logger == nil {
			return fmt.Errorf("cannot use nil logger")
		}

		option.logger = logger
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

// WithOTLPGrpcEndpoint set OpenTelemetry endpoint collector for type OTLP_GRPC span exporter using gRPC.
// Default to localhost:4317
func WithOTLPGrpcEndpoint(endpoint string) ExporterOpt {
	return func(option *ExporterOption) error {
		option.otlpGrpcEndpoint = endpoint
		return nil
	}
}

// WithHttpRoundTripper useful when we want to capture request-response log send by OpenTelemetry library.
// But please note to not add more tracing, you must only use this http.RoundTripper as logger only,
// or your tracing may create unwanted span if you add more span inside this middleware.
func WithHttpRoundTripper(r http.RoundTripper) ExporterOpt {
	return func(option *ExporterOption) error {
		option.httpRoundTripper = r
		return nil
	}
}

type ExporterOption struct {
	logger           *slog.Logger
	otlpEndpoint     string
	otlpGrpcEndpoint string
	httpRoundTripper http.RoundTripper
}

// NewTracerExporter select the tracer span exporter based on name.
// Default to noop exporter if no name or NOOP specified.
func NewTracerExporter(name string, opts ...ExporterOpt) (trace.SpanExporter, error) {
	cfg := &ExporterOption{
		logger:           slog.Default(),
		otlpEndpoint:     "localhost:4318",
		otlpGrpcEndpoint: "localhost:4317",
		httpRoundTripper: http.DefaultTransport,
	}

	for _, opt := range opts {
		err := opt(cfg)
		if err != nil {
			return nil, err
		}
	}

	httpClient := &http.Client{}
	httpClient.Transport = cfg.httpRoundTripper

	name = strings.TrimSpace(name)
	name = strings.ToUpper(name)
	switch name {
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

	case "OTLP_GRPC":
		endpoint := strings.TrimSpace(cfg.otlpGrpcEndpoint)
		if endpoint == "" {
			return nil, fmt.Errorf("cannot use OpenTelemetry OTLP_GRPC if OTEL_EXPORTER_OTLP_GRPC_ENDPOINT is empty")
		}

		return otlptrace.New(
			context.Background(),
			otlptracegrpc.NewClient(
				otlptracegrpc.WithInsecure(),
				otlptracegrpc.WithEndpoint(endpoint),
			),
		)

	case "STDOUT":
		return stdouttrace.New(
			stdouttrace.WithWriter(wrapToIO(cfg.logger)),
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

type slogIO struct {
	logger *slog.Logger
}

var _ io.Writer = (*slogIO)(nil)

func wrapToIO(logger *slog.Logger) *slogIO {
	return &slogIO{logger: logger}
}

func (s *slogIO) Write(p []byte) (n int, err error) {
	s.logger.InfoContext(context.Background(), string(p))
	return len(p), nil
}
