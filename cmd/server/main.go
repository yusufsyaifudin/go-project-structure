package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/caarlos0/env"
	_ "github.com/joho/godotenv/autoload"
	"github.com/yusufsyaifudin/go-project-structure/assets"
	"github.com/yusufsyaifudin/go-project-structure/internal/pkg/httpclientmw"
	"github.com/yusufsyaifudin/go-project-structure/internal/pkg/httpservermw"
	"github.com/yusufsyaifudin/go-project-structure/pkg/metrics"
	"github.com/yusufsyaifudin/go-project-structure/pkg/oteltracer"
	"github.com/yusufsyaifudin/go-project-structure/pkg/validator"
	"github.com/yusufsyaifudin/go-project-structure/pkg/ylog"
	"github.com/yusufsyaifudin/go-project-structure/transport/restapi"
	"github.com/yusufsyaifudin/go-project-structure/transport/restapi/handlersystem"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

type Config struct {
	HTTPPort        int    `env:"PORT" envDefault:"3000" validate:"required"`
	LogLevel        string `env:"LOG_LEVEL" envDefault:"DEBUG" validate:"required"`
	OtelExporter    string `env:"OTEL_EXPORTER" envDefault:"NOOP"` // NOOP, STDOUT, JAEGER, OTLP, OTLP_GRPC
	OtelOtlpURL     string `env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:"localhost:4318" validate:"required_if=OtelExporter OTLP"`
	OtelOtlpGrpcURL string `env:"OTEL_EXPORTER_OTLP_GRPC_ENDPOINT" envDefault:"localhost:4317" validate:"required_if=OtelExporter OTLP_GRPC"`
}

func main() {
	// systemCtx is context for system-wide process, it should not pass into HTTP or any Client process.
	systemCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	serviceName := assets.AppName + "_server"
	buildCommitID := assets.BuildCommitID()
	buildTime := assets.BuildTime()

	// *** Parse and validate config input
	cfg := Config{}
	err := env.Parse(&cfg)
	if err != nil {
		err = fmt.Errorf("cannot parse env var: %w", err)
		log.Fatalln(err)
		return
	}

	err = validator.Validate(cfg)
	if err != nil {
		err = fmt.Errorf("missing required config: %w", err)
		log.Fatalln(err)
		return
	}

	// ** Prepare logger using slog
	loggerOpt := &slog.HandlerOptions{
		AddSource:   true,
		Level:       slog.LevelDebug,
		ReplaceAttr: nil,
	}
	var loggerHandler slog.Handler = slog.NewJSONHandler(os.Stdout, loggerOpt)

	// ** Prepare logger using ylog
	yloggerOpt := &ylog.OpenTelemetryOption{
		ContextExtractor: func(ctx context.Context) []slog.Attr {
			return nil
		},
	}

	loggerHandler = ylog.NewOTEL(loggerHandler, yloggerOpt)
	logger := slog.New(loggerHandler)
	slog.SetDefault(logger)

	// Setup global error handler for OpenTelemetry SDK.
	// So, any error from OpenTelemetry will also comply with the standard slog.
	otel.SetErrorHandler(&otelErrHandler{})

	otelResource := newResource(systemCtx, serviceName)

	// prepare tracer exporter, whether using stdout or jaeger
	{
		tracerExporter, tracerExporterErr := oteltracer.NewTracerExporter(cfg.OtelExporter,
			oteltracer.WithLogger(slog.Default()),
			oteltracer.WithOTLPEndpoint(cfg.OtelOtlpURL),
			oteltracer.WithOTLPGrpcEndpoint(cfg.OtelOtlpGrpcURL),
			oteltracer.WithHttpRoundTripper(
				httpclientmw.NewHttpRoundTripper(
					httpclientmw.WithBaseRoundTripper(&http.Transport{}),
				),
			),
		)

		defer func() {
			if tracerExporter == nil {
				return
			}

			if _err := tracerExporter.Shutdown(systemCtx); _err != nil {
				slog.ErrorContext(systemCtx, "prepare exporter on shutdown error", slog.Any("error", _err))
			}
		}()

		if tracerExporterErr != nil {
			slog.ErrorContext(systemCtx, "prepare exporter error", slog.Any("error", tracerExporterErr))
			return
		}

		slog.ErrorContext(systemCtx, fmt.Sprintf("using %s as OpenTelemetry span exporter", cfg.OtelExporter))

		tracerProvider := trace.NewTracerProvider(
			trace.WithBatcher(tracerExporter),
			trace.WithResource(otelResource),
			trace.WithSampler(trace.AlwaysSample()),
		)
		defer func() {
			if _err := tracerProvider.Shutdown(systemCtx); _err != nil {
				slog.ErrorContext(systemCtx, "shutdown tracer error", slog.Any("error", _err))
			}
		}()

		// Set as global OpenTelemetry tracer provider.
		otel.SetTracerProvider(tracerProvider)
	}

	var combinedMetrics metrics.Metric
	{
		var prometheusMetric metrics.Metric
		prometheusMetric, err = metrics.NewPrometheus(
			metrics.PrometheusWithPrefix(serviceName + "_"),
		)
		if err != nil {
			slog.ErrorContext(systemCtx, "cannot prepare prometheus metric", slog.Any("error", err))
			return
		}

		// combine metrics to various type of outputs (prometheus, statsd, etc)
		combinedMetrics, err = metrics.NewCombinedMetrics(
			metrics.CombineMetricAdd(prometheusMetric),
		)
		if err != nil {
			slog.ErrorContext(systemCtx, "cannot combine metrics collector", slog.Any("error", err))
			return
		}
	}

	startupTime := time.Now()

	// prepare handler system for ping and system info routes.
	handlerSystem, err := handlersystem.New(
		handlersystem.WithBuildCommitID(buildCommitID),
		handlersystem.WithBuildTime(buildTime),
		handlersystem.WithStartupTime(startupTime),
	)
	if err != nil {
		slog.ErrorContext(systemCtx, "cannot prepare http handler for system router", slog.Any("error", err))
		return
	}

	// ** setup server with graceful shutdown
	slog.InfoContext(systemCtx, "preparing server http...")
	var serverMux http.Handler
	serverMux, err = restapi.NewHTTP(
		restapi.WithBuildCommitID(buildCommitID),
		restapi.WithBuildTime(buildTime),
		restapi.WithStartupTime(startupTime),

		// register all handler here
		restapi.AddHandler(handlerSystem),
	)
	if err != nil {
		slog.ErrorContext(systemCtx, "error prepare rest api server", slog.Any("error", err))
		return
	}

	// Register all endpoint that you won't need to be logged and traced.
	// For example, /ping can be skipped (return false) because it will be exhaust your Kubernetes log
	// if you set it as Readiness Probe.
	filterLogEndpoint := func(req *http.Request) bool {
		// Return "false" to indicate that this condition should be skipped in Log and Tracing.
		// Return "true" to indicate that this condition should be pushed in Log and Tracing.
		if req == nil {
			return true
		}

		if req.URL == nil {
			return true
		}

		switch strings.TrimRight(req.URL.Path, "/") {
		case "/favicon.ico", "/ping":
			return false
		}

		return true
	}

	// NOTE:
	// Please note that the HTTP raw middleware ordering is not like what we "naturally" think.
	// If we think that Logger middleware run before otelhttp middleware, you wrong!
	// The order of these middleware are:
	// 1. Remove trailing slash, then
	// 2. Add Prometheus middleware metrics, then
	// 3. Continue from request tracer span (if exist in request header) or create new tracer span, then
	// 4. Inject a non-exported span for filtered routes (so handler logs always carry trace_id), then
	// 5. Add middleware log!

	// Add logger middleware
	serverMux = httpservermw.LoggingMiddleware(serverMux,
		httpservermw.LogMwWithLogger(logger),
		httpservermw.LogMwWithTracer(otel.GetTracerProvider()),
		httpservermw.LogMwWithFilter(filterLogEndpoint),
	)

	// For routes filtered from otelhttp (e.g. /ping used as k8s readiness probe), otelhttp skips
	// span creation entirely, leaving a zero trace_id in any handler logs. SpanInjectorMiddleware
	// fills that gap: it starts a NeverSample span (real TraceID/SpanID, never exported) so that
	// slog.DebugContext and similar calls inside those handlers still produce meaningful trace context.
	serverMux = httpservermw.SpanInjectorMiddleware(serverMux, filterLogEndpoint)

	// Propagate OpenTelemetry tracing
	propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	serverMux = otelhttp.NewHandler(serverMux,
		assets.AppName+"_server",
		otelhttp.WithMessageEvents(otelhttp.ReadEvents, otelhttp.WriteEvents),
		otelhttp.WithPropagators(propagator),
		otelhttp.WithTracerProvider(otel.GetTracerProvider()),
		otelhttp.WithFilter(filterLogEndpoint),
		otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			if r == nil {
				return fmt.Sprintf("%s [on nil request]", operation)
			}

			if r.URL == nil {
				return fmt.Sprintf("%s %s [on nil url]", operation, r.Method)
			}

			return fmt.Sprintf("[%s] %s %s", operation, r.Method, r.URL.Path)
		}),
	)

	// Add Prometheus middleware metrics
	// prepare middleware and handler for Prometheus at the same time
	serverMux, err = httpservermw.PrometheusMiddleware(serverMux,
		httpservermw.PrometheusWithMetric(combinedMetrics),
	)
	if err != nil {
		slog.ErrorContext(systemCtx, "cannot prepare prometheus middleware", slog.Any("error", err))
		return
	}

	// Remove trailing slashes.
	serverMux = httpservermw.RemoveTrailingSlash(serverMux)

	httpPortStr := fmt.Sprintf(":%d", cfg.HTTPPort)

	// Enable HTTP/1.1, TLS HTTP/2, and cleartext HTTP/2 (h2c) using the Go 1.24+ Protocols field,
	// replacing the deprecated golang.org/x/net/http2/h2c package.
	var protocols http.Protocols
	protocols.SetHTTP1(true)
	protocols.SetHTTP2(true)
	protocols.SetUnencryptedHTTP2(true)

	httpServer := &http.Server{
		Addr:      httpPortStr,
		Handler:   serverMux,
		Protocols: &protocols,
	}

	var errChan = make(chan error, 1)
	go func() {
		slog.InfoContext(systemCtx, fmt.Sprintf("starting http on port %s", httpPortStr))
		errChan <- httpServer.ListenAndServe()
	}()

	var signalChan = make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	select {
	case s := <-signalChan:
		msg := fmt.Sprintf("got an interrupt: %+v", s)
		slog.ErrorContext(systemCtx, msg)
	case _err := <-errChan:
		if _err != nil {
			msg := fmt.Sprintf("error while running server: %s", _err)
			slog.ErrorContext(systemCtx, msg)
		}
	}
}

// newResource returns a resource describing this application.
func newResource(ctx context.Context, serviceName string) *resource.Resource {
	r := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
		semconv.ServiceVersion("v0.1.0"),
		semconv.DeploymentEnvironmentName("demo"),
	)

	if r == nil {
		slog.ErrorContext(ctx, "cannot use OpenTelemetry resource because of nil, fallback to default resource")
		return resource.Default()
	}

	return r
}

type otelErrHandler struct{}

var _ otel.ErrorHandler = (*otelErrHandler)(nil)

func (o *otelErrHandler) Handle(err error) {
	if err != nil {
		slog.ErrorContext(context.Background(), "OpenTelemetry error", slog.Any("error", err))
	}
}
