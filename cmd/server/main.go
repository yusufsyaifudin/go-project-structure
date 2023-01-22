package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/yusufsyaifudin/go-project-structure/internal/pkg/otel"

	"github.com/caarlos0/env"
	_ "github.com/joho/godotenv/autoload"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/yusufsyaifudin/go-project-structure/assets"
	"github.com/yusufsyaifudin/go-project-structure/internal/pkg/httpservermw"
	"github.com/yusufsyaifudin/go-project-structure/internal/pkg/observability"
	"github.com/yusufsyaifudin/go-project-structure/pkg/validator"
	"github.com/yusufsyaifudin/go-project-structure/pkg/ylog"
	"github.com/yusufsyaifudin/go-project-structure/transport/restapi"
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

	logger.Info(systemCtx, "trying to parse build time info")
	serverBuildTime := strings.TrimSpace(strings.Trim(assets.BuildTime, "\n"))
	buildTimeInt, err := strconv.Atoi(serverBuildTime)
	if err != nil {
		err = fmt.Errorf("BuildTime %+v variable not passed during build time: %w", assets.BuildTime, err)
		logger.Error(systemCtx, err.Error())
		return
	}

	buildTime := time.Unix(int64(buildTimeInt), 0)

	// prepare tracer exporter, whether using stdout or jaeger
	tracerExporter, tracerExporterErr := otel.NewTracerExporter(cfg.OtelExporter,
		otel.WithContext(systemCtx),
		otel.WithLogger(logger),
		otel.WithJaegerEndpoint(cfg.OtelJaegerURL),
		otel.WithOTLPEndpoint(cfg.OtelOtlpURL),
	)

	if tracerExporterErr != nil {
		logger.Error(systemCtx, "prepare exporter error", ylog.KV("error", tracerExporterErr.Error()))
		return
	}

	logger.Info(systemCtx, fmt.Sprintf("using %s as OpenTelemetry span exporter", cfg.OtelExporter))

	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(tracerExporter),
		trace.WithResource(newResource()),
		trace.WithSampler(trace.AlwaysSample()),
	)
	defer func() {
		if _err := tracerProvider.Shutdown(context.Background()); _err != nil {
			logger.Error(systemCtx, "shutdown tracer error", ylog.KV("error", _err))
		}
	}()

	const tracerName = "myapp-tracer"
	observeMgr, err := observability.NewManager(
		observability.WithLogger(logger),
		observability.WithTracerName(tracerName),
		observability.WithTracerProvider(tracerProvider),
	)
	if err != nil {
		logger.Error(systemCtx, "failed setup observability manager", ylog.KV("error", err))
		return
	}

	// ** setup server with graceful shutdown
	logger.Info(systemCtx, "preparing server http...")
	var serverMux http.Handler
	serverMux, err = restapi.NewHTTP(
		restapi.WithBuildCommitID(assets.BuildCommitID),
		restapi.WithBuildTime(buildTime),
		restapi.WithStartupTime(time.Now()),
		restapi.WithObservability(observeMgr),
	)
	if err != nil {
		err = fmt.Errorf("error prepare rest api server: %w", err)
		logger.Error(systemCtx, err.Error())
		return
	}

	serverMux = httpservermw.OpenTelemetryMiddleware(serverMux,
		httpservermw.OtelMwWithTracer(observeMgr.Tracer()),
	)

	// add logger middleware
	serverMux = httpservermw.LoggingMiddleware(serverMux,
		httpservermw.LogMwWithLogger(observeMgr.Logger()),
		httpservermw.LogMwWithMessage("incoming request log"),
		httpservermw.LogMwWithTracer(observeMgr.Tracer()),
	)

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
func newResource() *resource.Resource {
	r, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(assets.AppName),
			semconv.ServiceVersionKey.String("v0.1.0"),
			attribute.String("environment", "demo"),
		),
	)
	return r
}
