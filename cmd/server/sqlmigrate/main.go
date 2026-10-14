package sqlmigrate

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"time"

	"github.com/caarlos0/env/v11"
	_ "github.com/joho/godotenv/autoload" // by default, we will read .env before the real env vars
	_ "github.com/lib/pq"
	"github.com/hashicorp/cli"

	"github.com/yusufsyaifudin/go-project-structure/assets"
	"github.com/yusufsyaifudin/go-project-structure/pkg/validator"
)

const (
	exitCodeError = 1
)

type Config struct {
	PostgresDSN string `env:"POSTGRES_DSN" validate:"required"`
}

type Migration struct{}

var _ cli.Command = (*Migration)(nil)

func (m *Migration) Help() string {
	return "Database migration"
}

func (m *Migration) Run(args []string) int {
	systemCtx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

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

	buildCommitID := assets.BuildCommitID()
	c := cli.NewCLI("sql-migrate", buildCommitID)
	c.Args = args
	c.Commands = map[string]cli.CommandFactory{
		"up": func() (cli.Command, error) {
			return &Up{
				PostgresDSN: cfg.PostgresDSN,
			}, nil
		},
		"status": func() (cli.Command, error) {
			return &Status{
				PostgresDSN: cfg.PostgresDSN,
			}, nil
		},
	}

	exitStatus, err := c.Run()
	if err != nil {
		slog.ErrorContext(systemCtx, "error main program", slog.Any("error", err))
	}

	return exitStatus
}

func (m *Migration) Synopsis() string {
	return "Database migration"
}
