package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/caarlos0/env"
	_ "github.com/joho/godotenv/autoload"
	"github.com/mitchellh/cli"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"

	"github.com/yusufsyaifudin/go-project-structure/assets"
	pingcli "github.com/yusufsyaifudin/go-project-structure/cmd/cli/ping"
	"github.com/yusufsyaifudin/go-project-structure/internal/pkg/oteltracer"
	"github.com/yusufsyaifudin/go-project-structure/pkg/ylog"
)

type Config struct {
	LogLevel      string `env:"LOG_LEVEL" envDefault:"DEBUG"`
	OtelExporter  string `env:"OTEL_EXPORTER" envDefault:"NOOP"` // NOOP, STDOUT, JAEGER, OTLP
	OtelJaegerURL string `env:"OTEL_EXPORTER_JAEGER_ENDPOINT" envDefault:"http://localhost:14268/api/traces" validate:"required_if=OtelExporter JAEGER"`
	OtelOtlpURL   string `env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:"localhost:4318" validate:"required_if=OtelExporter OTLP"`
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

	// systemCtx is context for system-wide process, it should not pass into HTTP or any Client process.
	systemCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// ** Prepare logger using ylog
	logger := ylog.SetupZapLogger(cfg.LogLevel)

	// ** Prepare tracer for CLI (act as front-end).
	// This never block the CLI operation since it send through UDP.
	// If not configured, it will not export anything using noop exporter.
	var tracerExporter trace.SpanExporter
	var tracerErr error
	// prepare tracer exporter, whether using stdout or jaeger
	tracerExporter, tracerErr = oteltracer.NewTracerExporter(cfg.OtelExporter,
		oteltracer.WithContext(systemCtx),
		oteltracer.WithLogger(logger),
		oteltracer.WithJaegerEndpoint(cfg.OtelJaegerURL),
		oteltracer.WithOTLPEndpoint(cfg.OtelOtlpURL),
	)
	if tracerErr != nil {
		tracerExporter = tracetest.NewNoopExporter()

		logger.Error(systemCtx, "failed configure tracer", ylog.KV("error", err))
	} else {
		logger.Debug(systemCtx, fmt.Sprintf("using %s exporter", cfg.OtelExporter))
	}

	tracerProvider := trace.NewTracerProvider(
		// use sync operation to make sure every span persisted before CLI done
		trace.WithSyncer(tracerExporter),
		trace.WithResource(newResource()),
		trace.WithSampler(trace.AlwaysSample()),
	)
	defer func() {
		if _err := tracerProvider.Shutdown(context.Background()); _err != nil {
			logger.Error(systemCtx, "shutdown tracer error", ylog.KV("error", _err))
		}
	}()

	c := cli.NewCLI(assets.AppName, "1.0.0")
	c.Args = os.Args[1:]
	c.Commands = map[string]cli.CommandFactory{
		"ping": func() (cli.Command, error) {
			return pingcli.NewCMD(
				pingcli.WithTracer(tracerProvider),
				pingcli.WithLogger(logger),
			)
		},
	}

	exitStatus, err := c.Run()
	if err != nil {
		logger.Error(systemCtx, "failed to run cli", ylog.KV("error", err))
	}

	os.Exit(exitStatus)
}

// newResource returns a resource describing this application.
func newResource() *resource.Resource {
	r, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(assets.AppName+"_cli"),
			semconv.ServiceVersionKey.String("v0.1.0"),
			attribute.String("environment", "demo"),
		),
	)
	return r
}
