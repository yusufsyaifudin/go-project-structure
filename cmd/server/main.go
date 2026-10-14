package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/yusufsyaifudin/go-project-structure/cmd/server/sqlmigrate"

	"github.com/hashicorp/cli"

	"github.com/yusufsyaifudin/go-project-structure/cmd/server/server"
	"github.com/yusufsyaifudin/go-project-structure/pkg/ylog"
)

func main() {
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

	slog.Info("Loading server CLI...")

	c := cli.NewCLI("server", "1.0.0")
	c.Args = os.Args[1:]
	c.Commands = map[string]cli.CommandFactory{
		"": func() (cli.Command, error) {
			return &server.Server{}, nil
		},
		"server": func() (cli.Command, error) {
			return &server.Server{}, nil
		},
		"sql-migrate": func() (cli.Command, error) {
			return &sqlmigrate.Migration{}, nil
		},
	}

	exitStatus, err := c.Run()
	if err != nil {
		err = fmt.Errorf("error main program: %w", err)
		log.Println(err)
	}

	os.Exit(exitStatus)
}
