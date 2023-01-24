package oteltracer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/yusufsyaifudin/go-project-structure/pkg/ylog"
)

type ExporterOpt func(*expOption) error

func WithContext(ctx context.Context) ExporterOpt {
	return func(option *expOption) error {
		if ctx == nil {
			return nil
		}

		option.ctx = ctx
		return nil
	}
}

func WithLogger(logger ylog.Logger) ExporterOpt {
	return func(option *expOption) error {
		if logger == nil {
			return nil
		}

		option.logger = logger
		return nil
	}
}

func WithJaegerEndpoint(endpoint string) ExporterOpt {
	return func(option *expOption) error {
		if endpoint == "" {
			return nil
		}

		option.jaegerEndpoint = endpoint
		return nil
	}
}

func WithOTLPEndpoint(endpoint string) ExporterOpt {
	return func(option *expOption) error {
		if endpoint == "" {
			return nil
		}

		option.otlpEndpoint = endpoint
		return nil
	}
}

type expOption struct {
	ctx            context.Context
	logger         ylog.Logger
	jaegerEndpoint string
	otlpEndpoint   string
}

// NewTracerExporter select the tracer span exporter based on name.
// Default to noop exporter if no name or NOOP specified.
func NewTracerExporter(name string, opts ...ExporterOpt) (trace.SpanExporter, error) {
	cfg := &expOption{
		ctx:    context.Background(),
		logger: &ylog.Noop{},
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
			cfg.ctx,
			otlptracehttp.NewClient(
				otlptracehttp.WithInsecure(),
				otlptracehttp.WithEndpoint(endpoint),
			),
		)

	case "STDOUT":
		return stdouttrace.New(
			stdouttrace.WithWriter(debugWriter(cfg.ctx, cfg.logger)),
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

// loggerIOWriter wrap ylog.Logger as io.Writer
type loggerIOWriter struct {
	ctx    context.Context
	logger ylog.Logger
}

var _ io.Writer = (*loggerIOWriter)(nil)

// Write writes p as debug log using ylog.Logger.
// Since p may contain valid JSON object, we try to convert it as native Go object.
// Because if we write p directly to logger, it will print as Base64 encoded string.
// As a penalty, it may require some computation that not actually needed only to print the formatted JSON.
func (l *loggerIOWriter) Write(p []byte) (n int, err error) {
	var jsonObj interface{}
	if json.Unmarshal(p, &jsonObj) != nil {
		jsonObj = string(p)
	}

	l.logger.Debug(l.ctx, "tracer log", ylog.KV("data", jsonObj))
	return len(p), nil
}

// debugWriter wrap ylog.Logger as io.Writer with context.Context
func debugWriter(ctx context.Context, logger ylog.Logger) io.Writer {
	return &loggerIOWriter{
		ctx:    ctx,
		logger: logger,
	}
}
