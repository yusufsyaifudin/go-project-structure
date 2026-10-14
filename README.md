# go-project-structure

A minimal base project structure for Go HTTP services. It ships with a
working skeleton — HTTP server, Postgres wiring, SQL migrations,
OpenTelemetry tracing, Prometheus metrics, and a Docker Compose observability
stack — but no business domain. The only sample endpoints are `/ping`,
`/system-info`, and `/metrics`; replace or extend them with your own domain
handlers.

Key properties of the template:
- **HTTP framework**: Echo (`github.com/labstack/echo/v4`), with a
  pluggable `restapi.EchoRouter` interface for registering handler groups.
- **Postgres wiring**: `lib/pq` connection setup and `rubenv/sql-migrate`
  migrations, embedded via `go:embed`.
- **Built-in observability**: structured JSON logs, Prometheus metrics, and
  OpenTelemetry tracing (Jaeger / OTLP) included out of the box.
- **CLI structure**: `mitchellh/cli` sub-commands (`server`, `sql-migrate`).

## Quick start (Docker Compose)

The fastest way to run the service — no Go toolchain needed.

**1. Start the full stack:**

```shell
cd deployment/development
docker compose up -d
```

This brings up the service, Postgres, Jaeger, Prometheus, and Grafana.
The service is reachable at `http://localhost:3001`.

**2. Try the sample endpoints:**

```shell
curl -s http://localhost:3001/ping | jq .
curl -s http://localhost:3001/system-info | jq .
curl -s http://localhost:3001/metrics
```

## Local development

For active development where you run the service process directly on your machine:

**1. Start the backing services:**

```shell
cd deployment/development
docker compose up -d postgres
```

**2. Set up environment:**

```shell
cp .env.sample .env
# edit .env — verify POSTGRES_DSN is correct
```

**3. Run database migrations:**

```shell
go run cmd/server/main.go sql-migrate up
```

**4. Start the service:**

```shell
go run cmd/server/main.go
```

The service is reachable at `http://localhost:3001` (or whichever `PORT` you set in `.env`).

## Environment variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `PORT` | no | `3001` | HTTP listen port |
| `LOG_LEVEL` | no | `INFO` | `DEBUG` / `INFO` / `WARN` / `ERROR` |
| `POSTGRES_DSN` | **yes** | — | lib/pq DSN, e.g. `postgres://user:pass@host:5432/db?sslmode=disable` |
| `OTEL_EXPORTER` | no | `NOOP` | `NOOP` / `STDOUT` / `JAEGER` / `OTLP` / `OTLP_GRPC` |
| `OTEL_EXPORTER_JAEGER_ENDPOINT` | no | — | e.g. `http://localhost:14268/api/traces` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | no | — | host:port only, no scheme (HTTP) |
| `OTEL_EXPORTER_OTLP_GRPC_ENDPOINT` | no | — | host:port only, no scheme (gRPC) |

## Migrations

```shell
# Create a new migration file under migrations/
make migration-new name=add_something_useful

# Apply all pending migrations
POSTGRES_DSN="postgres://wallet:wallet_secret@localhost:5432/wallet_db?sslmode=disable" \
  ./server sql-migrate up

# Show migration status
POSTGRES_DSN="postgres://wallet:wallet_secret@localhost:5432/wallet_db?sslmode=disable" \
  ./server sql-migrate status
```

## Sample endpoints

These ship as a working reference for wiring your own handlers via the
`restapi.EchoRouter` interface (see `transport/restapi/handlersystem`):

- `GET /ping` — liveness check.
- `GET /system-info` — build commit, build time, startup time, uptime, Go runtime info.
- `GET /metrics` — Prometheus metrics exposition.

## Development

```shell
# Run tests
make test

# Lint
make lint

# Regenerate swagger docs (after editing swaggo annotations)
make swagger

# Build binary
go build -o server ./cmd/server/main.go

# Create a new migration file
make migration-new name=add_something_useful
```

## Project structure

```
cmd/
  server/            — server binary entrypoint (HTTP server, sql-migrate CLI)
  cli/                — standalone CLI binary (sample "ping" sub-command)
assets/               — embedded build metadata (commit hash, build time)
internal/
  pkg/
    httpclientmw/     — outbound HTTP client middleware
    httpservermw/     — inbound HTTP server middleware (access log, Prometheus, span injector, trailing slash)
pkg/                  — shared, reusable utilities
  distlock/           — generic distributed-lock interface + in-process implementation
  idgen/              — Sonyflake ID generator
  metrics/            — Prometheus + noop metrics
  oteltracer/         — OpenTelemetry tracer/exporter setup
  respbuilder/        — uniform JSON envelope (Ok/Error)
  sqldb/              — SQL DB helpers
  validator/          — go-playground/validator wrapper + IsStrictUUIDv4
  ylog/               — slog + OpenTelemetry attribute injection
transport/restapi/    — Echo HTTP server setup and EchoRouter registration
  handlersystem/      — /ping, /system-info handlers (sample)
migrations/           — sql-migrate SQL files (YYYY-MM-DD-HH-mm-SS_name.sql)
deployment/           — Docker Compose (development) and Kubernetes manifests
docs/                 — swagger output, RFC template location
```

## Rename the module and packages

When you fork this template, update the Go module path and the project’s
package references with the included Bash script:

```shell
./scripts/rename-module.sh \
  github.com/yusufsyaifudin/go-project-structure \
  github.com/your-org/your-service \
  go-project-structure \
  your-service
```

The arguments are the old module path, new module path, old project name, and
new project name. To rename a Go package identifier as well, append its old
and new identifiers as arguments. The script replaces module references in
tracked text files, including `go.mod`, Go imports and package instrumentation names. It skips
`.git`, generated Swagger output, and dependency lock data. Review the diff,
then run `go mod tidy` after the rename. Use lowercase kebab-case for project
names; Go package identifiers should remain lowercase words without hyphens.
