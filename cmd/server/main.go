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

	"github.com/caarlos0/env"
	"github.com/google/uuid"
	_ "github.com/joho/godotenv/autoload"
	"github.com/yusufsyaifudin/go-project-structure/assets"
	"github.com/yusufsyaifudin/go-project-structure/pkg/validator"
	"github.com/yusufsyaifudin/go-project-structure/transport/restapi"
	"github.com/yusufsyaifudin/ylog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	HTTPPort int `env:"PORT" envDefault:"3000" validate:"required"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	type CtxDataSystem struct {
		RequestID     string `tracer:"trace_id"`
		CorrelationID string `tracer:"correlation_id"`
	}

	propagateData := CtxDataSystem{
		RequestID:     "system",
		CorrelationID: uuid.NewString(),
	}

	tracer, err := ylog.NewTracer(propagateData, ylog.WithTag("tracer"))
	if err != nil {
		err = fmt.Errorf("system context data: %w", err)
		log.Fatalln(err)
		return
	}

	ctx = ylog.Inject(ctx, tracer)

	// *** Parse and validate config input
	cfg := Config{}
	err = env.Parse(&cfg)
	if err != nil {
		err = fmt.Errorf("cannot parse env var: %w", err)
		ylog.Error(ctx, err.Error())
		return
	}

	err = validator.Validate(cfg)
	if err != nil {
		err = fmt.Errorf("missing required config: %w", err)
		ylog.Error(ctx, err.Error())
		return
	}

	// ** Prepare logger
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zapcore.EncoderConfig{
			TimeKey:        "ts",
			MessageKey:     "msg",
			EncodeDuration: zapcore.MillisDurationEncoder,
			EncodeTime:     zapcore.RFC3339NanoTimeEncoder,
			LineEnding:     zapcore.DefaultLineEnding,
			LevelKey:       "level",
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
		}),
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout)), // pipe to multiple writer
		zapcore.DebugLevel,
	)

	zapLogger := zap.New(core)
	uniLogger := ylog.NewZap(zapLogger)

	// set to global logger
	ylog.SetGlobalLogger(uniLogger)

	ylog.Info(ctx, "trying to parse build time info")
	serverBuildTime := strings.TrimSpace(strings.Trim(assets.BuildTime, "\n"))
	buildTimeInt, err := strconv.Atoi(serverBuildTime)
	if err != nil {
		err = fmt.Errorf("BuildTime %+v variable not passed during build time: %w", assets.BuildTime, err)
		ylog.Error(ctx, err.Error())
		return
	}

	buildTime := time.Unix(int64(buildTimeInt), 0)

	// ** setup server with graceful shutdown
	ylog.Info(ctx, "preparing server http...")
	serverMuxCfg := restapi.HTTPConfig{
		BuildCommitID: assets.BuildCommitID,
		BuildTime:     buildTime,
		StartupTime:   time.Now(),
	}

	serverMux, err := restapi.NewHTTP(serverMuxCfg)
	if err != nil {
		err = fmt.Errorf("error prepare rest api server: %w", err)
		ylog.Error(ctx, err.Error())
		return
	}

	var errChan = make(chan error, 1)
	go func() {
		httpPortStr := fmt.Sprintf(":%d", cfg.HTTPPort)
		ylog.Info(ctx, fmt.Sprintf("starting http on port %s", httpPortStr))
		if _err := http.ListenAndServe(httpPortStr, serverMux); _err != nil {
			errChan <- fmt.Errorf("http server error: %w", _err)
		}
	}()

	var signalChan = make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	select {
	case s := <-signalChan:
		msg := fmt.Sprintf("got an interrupt: %+v", s)
		ylog.Error(ctx, msg)
	case _err := <-errChan:
		if _err != nil {
			msg := fmt.Sprintf("error while running server: %s", _err)
			ylog.Error(ctx, msg)
		}
	}
}
