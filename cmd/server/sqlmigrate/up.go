package sqlmigrate

import (
	"database/sql"
	"log/slog"

	"github.com/yusufsyaifudin/go-project-structure/migrations"

	"github.com/mitchellh/cli"
	migrate "github.com/rubenv/sql-migrate"
)

type Up struct {
	PostgresDSN string
}

var _ cli.Command = (*Up)(nil)

func (u *Up) Help() string {
	return "Apply all available migrations"
}

func (u *Up) Run(args []string) int {
	const dialect = "postgres"
	db, dbErr := sql.Open(dialect, u.PostgresDSN)
	if dbErr != nil {
		slog.Error("failed to connect to database", slog.String("error", dbErr.Error()))
		return 1
	}
	defer func() {
		if _err := db.Close(); _err != nil {
			slog.Error("failed to close database connection", slog.String("error", _err.Error()))
		}
	}()

	migrationsFiles := migrate.EmbedFileSystemMigrationSource{
		FileSystem: migrations.FS,
		Root:       ".",
	}

	migrate.SetTable("schema_migrations")
	upMigrate, upMigrateErr := migrate.Exec(db, dialect, migrationsFiles, migrate.Up)
	if upMigrateErr != nil {
		slog.Error("failed to apply migrations", slog.String("error", upMigrateErr.Error()))
		return 1
	}

	slog.Info("migrations applied successfully", slog.Int("count", upMigrate))

	status := &Status{
		PostgresDSN: u.PostgresDSN,
	}

	return status.Run(args)
}

func (u *Up) Synopsis() string {
	return "Apply all available migrations"
}
