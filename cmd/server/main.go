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

	_ "github.com/joho/godotenv/autoload"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/caarlos0/env"
	"github.com/yusufsyaifudin/go-project-structure/assets"
	"github.com/yusufsyaifudin/go-project-structure/internal/pkg/httpservermw"
	"github.com/yusufsyaifudin/go-project-structure/pkg/validator"
	"github.com/yusufsyaifudin/go-project-structure/pkg/ylog"
	"github.com/yusufsyaifudin/go-project-structure/transport/restapi"
)

type Config struct {
	HTTPPort int    `env:"PORT" envDefault:"3000" validate:"required"`
	LogLevel string `env:"LOG_LEVEL" envDefault:"DEBUG" validate:"required"`
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
	ylog.SetupDefaultGlobalLogger(cfg.LogLevel)

	ylog.Info(systemCtx, "trying to parse build time info")
	serverBuildTime := strings.TrimSpace(strings.Trim(assets.BuildTime, "\n"))
	buildTimeInt, err := strconv.Atoi(serverBuildTime)
	if err != nil {
		err = fmt.Errorf("BuildTime %+v variable not passed during build time: %w", assets.BuildTime, err)
		ylog.Error(systemCtx, err.Error())
		return
	}

	buildTime := time.Unix(int64(buildTimeInt), 0)

	// ** setup server with graceful shutdown
	ylog.Info(systemCtx, "preparing server http...")
	serverMuxCfg := restapi.HTTPConfig{
		BuildCommitID: assets.BuildCommitID,
		BuildTime:     buildTime,
		StartupTime:   time.Now(),
	}

	var serverMux http.Handler
	serverMux, err = restapi.NewHTTP(serverMuxCfg)
	if err != nil {
		err = fmt.Errorf("error prepare rest api server: %w", err)
		ylog.Error(systemCtx, err.Error())
		return
	}

	// add logger middleware
	serverMux = httpservermw.LoggingMiddleware(serverMux,
		httpservermw.WithLogger(ylog.GetGlobalLogger()),
		httpservermw.WithMessage("incoming request log"),
	)

	httpPortStr := fmt.Sprintf(":%d", cfg.HTTPPort)
	httpServer := &http.Server{
		Addr:    httpPortStr,
		Handler: h2c.NewHandler(serverMux, &http2.Server{}), // HTTP/2 Cleartext handler
	}

	var errChan = make(chan error, 1)
	go func() {
		ylog.Info(systemCtx, fmt.Sprintf("starting http on port %s", httpPortStr))
		errChan <- httpServer.ListenAndServe()
	}()

	var signalChan = make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	select {
	case s := <-signalChan:
		msg := fmt.Sprintf("got an interrupt: %+v", s)
		ylog.Error(systemCtx, msg)
	case _err := <-errChan:
		if _err != nil {
			msg := fmt.Sprintf("error while running server: %s", _err)
			ylog.Error(systemCtx, msg)
		}
	}
}
