package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/caarlos0/env"
	_ "github.com/joho/godotenv/autoload"
	"github.com/mitchellh/cli"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"

	"github.com/yusufsyaifudin/go-project-structure/assets"
	pingcli "github.com/yusufsyaifudin/go-project-structure/cmd/cli/ping"
	"github.com/yusufsyaifudin/go-project-structure/pkg/oteltracer"
	"github.com/yusufsyaifudin/go-project-structure/pkg/ylog"
)

type Config struct {
	LogLevel        string `env:"LOG_LEVEL" envDefault:"DEBUG"`
	OtelExporter    string `env:"OTEL_EXPORTER" envDefault:"NOOP"` // NOOP, STDOUT, JAEGER, OTLP, OTLP_GRPC
	OtelOtlpURL     string `env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:"localhost:4318" validate:"required_if=OtelExporter OTLP"`
	OtelOtlpGrpcURL string `env:"OTEL_EXPORTER_OTLP_GRPC_ENDPOINT" envDefault:"localhost:4317" validate:"required_if=OtelExporter OTLP_GRPC"`
}

func main() {
	// *** Parse and validate config input
	cfg := Config{}
	err := env.Parse(&cfg)
	if err != nil {
		err = fmt.Errorf("cannot parse env var: %w", err)
		log.Fatalln(err)
		return
	}

	serviceName := assets.AppName + "_cli"

	// systemCtx is context for system-wide process, it should not pass into HTTP or any Client process.
	systemCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

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

	otelResources := newResource(systemCtx, serviceName)

	// ** Prepare tracer for CLI (act as front-end).
	// This never block the CLI operation since it send through UDP.
	// If not configured, it will not export anything using noop exporter.
	{
		var tracerExporter trace.SpanExporter
		var tracerErr error

		// prepare tracer exporter, whether using stdout or jaeger
		tracerExporter, tracerErr = oteltracer.NewTracerExporter(cfg.OtelExporter,
			oteltracer.WithLogger(slog.Default()),
			oteltracer.WithOTLPEndpoint(cfg.OtelOtlpURL),
			oteltracer.WithOTLPGrpcEndpoint(cfg.OtelOtlpGrpcURL),
		)
		if tracerErr != nil {
			tracerExporter = tracetest.NewNoopExporter()

			slog.ErrorContext(systemCtx, "failed configure tracer", slog.Any("error", err))
		} else {
			slog.InfoContext(systemCtx, fmt.Sprintf("using %s exporter", cfg.OtelExporter))
		}

		tracerProviderImplemented := trace.NewTracerProvider(
			// use sync operation to make sure every span persisted before CLI done
			trace.WithSyncer(tracerExporter),
			trace.WithResource(otelResources),
			trace.WithSampler(trace.AlwaysSample()),
		)
		defer func() {
			if _err := tracerProviderImplemented.Shutdown(systemCtx); _err != nil {
				slog.ErrorContext(systemCtx, "shutdown tracer error", slog.Any("error", _err))
			}
		}()

		otel.SetTracerProvider(tracerProviderImplemented)
	}

	c := cli.NewCLI(assets.AppName, "1.0.0")
	c.Args = os.Args[1:]
	c.Commands = map[string]cli.CommandFactory{
		"ping": func() (cli.Command, error) {
			return pingcli.NewCMD(
				pingcli.WithTracer(otel.GetTracerProvider()),
			)
		},
	}

	exitStatus, err := c.Run()
	if err != nil {
		slog.ErrorContext(systemCtx, "failed to run cli", slog.Any("error", err))
	}

	os.Exit(exitStatus)
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
