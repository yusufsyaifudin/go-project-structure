package sqlmigrate

import (
	"database/sql"
	"log/slog"
	"os"
	"time"

	"github.com/yusufsyaifudin/go-project-structure/migrations"

	"github.com/mitchellh/cli"
	"github.com/olekukonko/tablewriter"
	migrate "github.com/rubenv/sql-migrate"
)

type Status struct {
	PostgresDSN string
}

var _ cli.Command = (*Status)(nil)

func (u *Status) Help() string {
	return "See current migration status"
}

func (u *Status) Run([]string) int {
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
	records, recordsErr := migrate.GetMigrationRecords(db, dialect)
	if recordsErr != nil {
		slog.Error("failed to get migration records", slog.String("error", recordsErr.Error()))
		return 1
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"Migration", "Applied"})

	type statusRow struct {
		Id        string
		Migrated  bool
		AppliedAt time.Time
	}
	rows := make(map[string]*statusRow)

	migrationSources, migrationSourcesErr := migrationsFiles.FindMigrations()
	if migrationSourcesErr != nil {
		slog.Error("failed to find migrations", slog.String("error", migrationSourcesErr.Error()))
		return 1
	}

	for _, m := range migrationSources {
		rows[m.Id] = &statusRow{
			Id:       m.Id,
			Migrated: false,
		}
	}

	for _, r := range records {
		if rows[r.Id] == nil {
			slog.Warn("migration record found without corresponding file", slog.String("migration_id", r.Id))
			continue
		}

		rows[r.Id].Migrated = true
		rows[r.Id].AppliedAt = r.AppliedAt
	}

	for _, m := range migrationSources {
		if rows[m.Id] != nil && rows[m.Id].Migrated {
			_ = table.Append([]string{
				m.Id,
				rows[m.Id].AppliedAt.String(),
			})
		} else {
			_ = table.Append([]string{
				m.Id,
				"no",
			})
		}
	}

	_ = table.Render()
	return 0
}

func (u *Status) Synopsis() string {
	return "See current migration status"
}
