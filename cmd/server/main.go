package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/caarlos0/env"
	_ "github.com/joho/godotenv/autoload"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/yusufsyaifudin/go-project-structure/assets"
	"github.com/yusufsyaifudin/go-project-structure/internal/pkg/httpservermw"
	"github.com/yusufsyaifudin/go-project-structure/internal/pkg/observability"
	"github.com/yusufsyaifudin/go-project-structure/pkg/oteltracer"
	"github.com/yusufsyaifudin/go-project-structure/pkg/validator"
	"github.com/yusufsyaifudin/go-project-structure/pkg/ylog"
	"github.com/yusufsyaifudin/go-project-structure/transport/restapi"
	"github.com/yusufsyaifudin/go-project-structure/transport/restapi/handlersystem"
)

type Config struct {
	HTTPPort      int    `env:"PORT" envDefault:"3000" validate:"required"`
	LogLevel      string `env:"LOG_LEVEL" envDefault:"DEBUG" validate:"required"`
	OtelExporter  string `env:"OTEL_EXPORTER" envDefault:"NOOP"` // NOOP, STDOUT, JAEGER, OTLP
	OtelJaegerURL string `env:"OTEL_EXPORTER_JAEGER_ENDPOINT" envDefault:"http://localhost:14268/api/traces" validate:"required_if=OtelExporter JAEGER"`
	OtelOtlpURL   string `env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:"localhost:4318" validate:"required_if=OtelExporter OTLP"`
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

	// ** Prepare logger using ylog
	logger := ylog.SetupZapLogger(cfg.LogLevel)

	// prepare Prometheus registry
	prometheusRegistry := prometheus.NewRegistry()

	// prepare tracer exporter, whether using stdout or jaeger
	tracerExporter, tracerExporterErr := oteltracer.NewTracerExporter(cfg.OtelExporter,
		oteltracer.WithLogger(ylog.WrapIOWriter(logger)),
		oteltracer.WithJaegerEndpoint(cfg.OtelJaegerURL),
		oteltracer.WithOTLPEndpoint(cfg.OtelOtlpURL),
	)

	if tracerExporterErr != nil {
		logger.Error(systemCtx, "prepare exporter error", ylog.KV("error", tracerExporterErr.Error()))
		return
	}

	logger.Info(systemCtx, fmt.Sprintf("using %s as OpenTelemetry span exporter", cfg.OtelExporter))

	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(tracerExporter),
		trace.WithResource(newResource(serviceName)),
		trace.WithSampler(trace.AlwaysSample()),
	)
	defer func() {
		if _err := tracerProvider.Shutdown(context.Background()); _err != nil {
			logger.Error(systemCtx, "shutdown tracer error", ylog.KV("error", _err))
		}
	}()

	observeMgr, err := observability.NewManager(
		observability.WithLogger(logger),
		observability.WithTracerProvider(tracerProvider),
	)
	if err != nil {
		logger.Error(systemCtx, "failed setup observability manager", ylog.KV("error", err))
		return
	}

	startupTime := time.Now()

	// prepare handler system for ping and system info routes.
	handlerSystem, err := handlersystem.New(
		handlersystem.WithBuildCommitID(buildCommitID),
		handlersystem.WithBuildTime(buildTime),
		handlersystem.WithStartupTime(startupTime),
		handlersystem.WithObservability(observeMgr),
	)
	if err != nil {
		logger.Error(systemCtx, "cannot prepare http handler for system router", ylog.KV("error", err))
		return
	}

	// ** setup server with graceful shutdown
	logger.Info(systemCtx, "preparing server http...")
	var serverMux http.Handler
	serverMux, err = restapi.NewHTTP(
		restapi.WithBuildCommitID(buildCommitID),
		restapi.WithBuildTime(buildTime),
		restapi.WithStartupTime(startupTime),
		restapi.WithObservability(observeMgr),

		// register all handler here
		restapi.AddHandler(handlerSystem),
	)
	if err != nil {
		err = fmt.Errorf("error prepare rest api server: %w", err)
		logger.Error(systemCtx, err.Error())
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
		case "/metrics":
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
	// 4. Add middleware log!

	// Add logger middleware
	serverMux = httpservermw.LoggingMiddleware(serverMux,
		httpservermw.LogMwWithLogger(logger),
		httpservermw.LogMwWithMessage("incoming request log"),
		httpservermw.LogMwWithTracer(tracerProvider),
		httpservermw.LogMwWithFilter(filterLogEndpoint),
	)

	// Propagate OpenTelemetry tracing
	propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	serverMux = otelhttp.NewHandler(serverMux,
		assets.AppName+"_server",
		otelhttp.WithMessageEvents(otelhttp.ReadEvents, otelhttp.WriteEvents),
		otelhttp.WithPropagators(propagator),
		otelhttp.WithTracerProvider(tracerProvider),
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
		httpservermw.PrometheusOptEnableGoMetric(true),
		// Add prefix "servicename_" to differentiate from another metric
		httpservermw.PrometheusOptWithRegisterer(prometheus.WrapRegistererWithPrefix(serviceName+"_", prometheusRegistry)),
		httpservermw.PrometheusOptWithGatherer(prometheusRegistry),
	)
	if err != nil {
		logger.Error(systemCtx, "cannot prepare prometheus middleware", ylog.KV("error", err))
		return
	}

	// Remove trailing slashes.
	serverMux = httpservermw.RemoveTrailingSlash(serverMux)

	httpPortStr := fmt.Sprintf(":%d", cfg.HTTPPort)
	httpServer := &http.Server{
		Addr:    httpPortStr,
		Handler: h2c.NewHandler(serverMux, &http2.Server{}), // HTTP/2 Cleartext handler
	}

	var errChan = make(chan error, 1)
	go func() {
		logger.Info(systemCtx, fmt.Sprintf("starting http on port %s", httpPortStr))
		errChan <- httpServer.ListenAndServe()
	}()

	var signalChan = make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	select {
	case s := <-signalChan:
		msg := fmt.Sprintf("got an interrupt: %+v", s)
		logger.Error(systemCtx, msg)
	case _err := <-errChan:
		if _err != nil {
			msg := fmt.Sprintf("error while running server: %s", _err)
			logger.Error(systemCtx, msg)
		}
	}
}

// newResource returns a resource describing this application.
func newResource(serviceName string) *resource.Resource {
	r, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String("v0.1.0"),
			attribute.String("environment", "demo"),
		),
	)
	return r
}
