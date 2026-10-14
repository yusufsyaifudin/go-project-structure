package server

import (
	"context"
	"database/sql"
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

	"github.com/caarlos0/env/v11"
	_ "github.com/joho/godotenv/autoload"
	_ "github.com/lib/pq"
	"github.com/hashicorp/cli"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"

	"github.com/yusufsyaifudin/go-project-structure/assets"
	_ "github.com/yusufsyaifudin/go-project-structure/docs/swagger" // registers generated swagger spec
	"github.com/yusufsyaifudin/go-project-structure/internal/pkg/httpclientmw"
	"github.com/yusufsyaifudin/go-project-structure/internal/pkg/httpservermw"
	"github.com/yusufsyaifudin/go-project-structure/pkg/metrics"
	"github.com/yusufsyaifudin/go-project-structure/pkg/oteltracer"
	"github.com/yusufsyaifudin/go-project-structure/pkg/validator"
	"github.com/yusufsyaifudin/go-project-structure/pkg/ylog"
	"github.com/yusufsyaifudin/go-project-structure/transport/restapi"
	"github.com/yusufsyaifudin/go-project-structure/transport/restapi/handlersystem"
)

const (
	exitCodeError = 1
	exitCodeOK    = 0
)

type Config struct {
	HTTPPort        int    `env:"PORT" envDefault:"3000" validate:"required"`
	BaseURL         string `env:"BASE_URL" envDefault:"http://localhost:3000/" validate:"required"`
	LogLevel        string `env:"LOG_LEVEL" envDefault:"DEBUG" validate:"required"`
	Environment     string `env:"APP_ENV" envDefault:"demo" validate:"required"` // e.g. demo, staging, production
	OtelExporter    string `env:"OTEL_EXPORTER" envDefault:"NOOP"`               // NOOP, STDOUT, JAEGER, OTLP, OTLP_GRPC
	OtelOtlpURL     string `env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:"localhost:4318" validate:"required_if=OtelExporter OTLP"`
	OtelOtlpGrpcURL string `env:"OTEL_EXPORTER_OTLP_GRPC_ENDPOINT" envDefault:"localhost:4317" validate:"required_if=OtelExporter OTLP_GRPC"`
	PostgresDSN     string `env:"POSTGRES_DSN" validate:"required"`
}

type Server struct{}

var _ cli.Command = (*Server)(nil)

func (s *Server) Help() string {
	return ""
}

func (s *Server) Synopsis() string {
	return "Start HTTP server"
}

func (s *Server) Run(_ []string) int {
	// systemCtx is context for system-wide process, it should not pass into HTTP or any Client process.
	systemCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	serviceName := assets.AppName + "_server"
	buildCommitID := assets.BuildCommitID()
	buildTime := assets.BuildTime()

	// *** Parse and validate config input
	cfg := Config{}
	if _err := env.Parse(&cfg); _err != nil {
		_err = fmt.Errorf("cannot parse env var: %w", _err)
		log.Fatalln(_err)
		return exitCodeError
	}

	if _err := validator.Validate(cfg); _err != nil {
		_err = fmt.Errorf("missing required config: %w", _err)
		log.Fatalln(_err)
		return exitCodeError
	}

	// ** Prepare logger using slog
	{
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
	}

	// Setup global error handler for OpenTelemetry SDK.
	otel.SetErrorHandler(&otelErrHandler{})

	otelResource := newResource(systemCtx, serviceName, buildCommitID, cfg.Environment)

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
			return exitCodeError
		}

		slog.InfoContext(systemCtx, fmt.Sprintf("using %s as OpenTelemetry span exporter", cfg.OtelExporter))

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
		var metricErr error
		var prometheusMetric metrics.Metric
		prometheusMetric, metricErr = metrics.NewPrometheus(
			metrics.PrometheusWithPrefix(serviceName + "_"),
		)
		if metricErr != nil {
			slog.ErrorContext(systemCtx, "cannot prepare prometheus metric", slog.Any("error", metricErr))
			return exitCodeError
		}

		// combine metrics to various type of outputs (prometheus, statsd, etc)
		combinedMetrics, metricErr = metrics.NewCombinedMetrics(
			metrics.CombineMetricAdd(prometheusMetric),
		)
		if metricErr != nil {
			slog.ErrorContext(systemCtx, "cannot combine metrics collector", slog.Any("error", metricErr))
			return exitCodeError
		}
	}

	// *** Connect to Postgres
	db, dbErr := sql.Open("postgres", cfg.PostgresDSN)
	if dbErr != nil {
		slog.ErrorContext(systemCtx, "cannot open postgres connection", slog.Any("error", dbErr))
		return exitCodeError
	}
	defer func() {
		if _err := db.Close(); _err != nil {
			slog.ErrorContext(systemCtx, "postgres close error", slog.Any("error", _err))
		}
	}()

	if _err := db.PingContext(systemCtx); _err != nil {
		slog.ErrorContext(systemCtx, "postgres ping failed", slog.Any("error", _err))
		return exitCodeError
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
		return exitCodeError
	}

	// ** setup server with graceful shutdown
	slog.InfoContext(systemCtx, "preparing server http...")
	var serverMux http.Handler
	serverMux, err = restapi.NewHTTP(
		restapi.WithBuildCommitID(buildCommitID),
		restapi.WithBuildTime(buildTime),
		restapi.WithStartupTime(startupTime),
		restapi.WithBaseURL(cfg.BaseURL),

		// register all handler here
		restapi.AddHandler(handlerSystem),
	)
	if err != nil {
		slog.ErrorContext(systemCtx, "error prepare rest api server", slog.Any("error", err))
		return exitCodeError
	}

	filterLogEndpoint := func(req *http.Request) bool {
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

	// Add logger middleware
	serverMux = httpservermw.LoggingMiddleware(serverMux,
		httpservermw.LogMwWithLogger(slog.Default()),
		httpservermw.LogMwWithTracer(otel.GetTracerProvider()),
		httpservermw.LogMwWithFilter(filterLogEndpoint),
	)

	serverMux = httpservermw.SpanInjectorMiddleware(serverMux, filterLogEndpoint)

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
	serverMux, err = httpservermw.PrometheusMiddleware(serverMux,
		httpservermw.PrometheusWithMetric(combinedMetrics),
	)
	if err != nil {
		slog.ErrorContext(systemCtx, "cannot prepare prometheus middleware", slog.Any("error", err))
		return exitCodeError
	}

	// Remove trailing slashes.
	serverMux = httpservermw.RemoveTrailingSlash(serverMux)

	httpPortStr := fmt.Sprintf(":%d", cfg.HTTPPort)

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
	case sign := <-signalChan:
		msg := fmt.Sprintf("got an interrupt: %+v", sign)
		slog.ErrorContext(systemCtx, msg)
	case _err := <-errChan:
		if _err != nil {
			msg := fmt.Sprintf("error while running server: %s", _err)
			slog.ErrorContext(systemCtx, msg)
		}
	}

	return exitCodeOK
}

// newResource returns a resource describing this application.
// version should be assets.BuildCommitID(); environment comes from Config.Environment
// for the HTTP server and is left as "demo" for CLI subcommands that don't need it.
func newResource(ctx context.Context, serviceName, version, environment string) *resource.Resource {
	r := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
		semconv.ServiceVersion(version),
		semconv.DeploymentEnvironmentName(environment),
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
		// WARN not ERROR: export failures are transient and retried internally by the OTel SDK.
		// Logging at ERROR implies the process is broken, which it isn't — the backend is just
		// unreachable or slow, and the SDK will retry on the next flush interval.
		slog.WarnContext(context.Background(), "OpenTelemetry error", slog.Any("error", err))
	}
}
