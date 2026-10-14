# AGENT.md – Coding Standards for go-project-structure

## Module

`github.com/yusufsyaifudin/go-project-structure`
Go version: **1.27.2** (see `go.mod`).

This repository is a **base project template**: a working Go HTTP service
skeleton (HTTP server, Postgres wiring, migrations, tracing, metrics) with
no business domain. The `/ping` and `/system-info` handlers under
`handlersystem` are the sample reference implementation for wiring new
handler groups.

---

## Project Layout

```
cmd/
  server/             – binary entry points and CLI sub-commands
    main.go           – wires hashicorp/cli commands
    server/main.go    – HTTP server command (Config, Run)
    sqlmigrate/       – sql-migrate wrapper commands (up, status)
  cli/                – standalone CLI binary (sample "ping" sub-command)
assets/               – embedded build metadata (commit hash, build time)
migrations/           – SQL files embedded via assets.go (rubenv/sql-migrate format)
internal/
  pkg/
    httpclientmw/     – outbound HTTP middleware
    httpservermw/     – inbound HTTP middleware
pkg/
  distlock/           – generic Locker interface + in-process (LocalLocker) implementation
  idgen/              – Sonyflake ID generator (int64, application-assigned)
  metrics/            – Prometheus + noop metrics
  oteltracer/         – OpenTelemetry tracer exporter setup
  respbuilder/        – uniform JSON envelope (Ok/Error)
  sqldb/              – SQL DB helpers
  validator/          – go-playground/validator wrapper + IsStrictUUIDv4
  ylog/               – slog + OpenTelemetry attribute injection
transport/
  restapi/            – Echo HTTP server setup and EchoRouter registration
    handlersystem/    – /ping, /system-info handlers (sample reference implementation)
docs/rfc/             – approved RFC documents (read-only after approval)
```

---

## Naming & Style

- Follow standard Go conventions (https://go.dev/doc/effective_go).
- Package names: lowercase, no underscores, single word where possible.
- Exported types/functions: PascalCase. Unexported: camelCase.
- Error variables: `Err` prefix (e.g. `distlock.ErrLockHeld`).
- Interface names: suffix with the role, not `I`-prefix (`Locker`, not `ILocker`).
- Constructor pattern: `New(opts ...Opt) (*T, error)` for complex types; plain struct literals for simple ones.
- Functional options: `type Opt func(*T) error` pattern — see `handlersystem` for a reference.
- HTTP handler types: `type Opt func(*Handler) error` + `New(opts ...Opt) (*Handler, error)`.

---

## Error Handling

- **Never swallow errors.** Always propagate or log with context.
- Use sentinel errors (plain `errors.New`) for conditions that callers must distinguish (e.g. `distlock.ErrLockHeld`).
- Use `fmt.Errorf("...: %w", err)` to wrap with context.
- HTTP handlers check sentinel errors with `errors.Is` and map them to the correct `respbuilder` codes.

---

## Response Envelope

**All HTTP responses** must go through `pkg/respbuilder`:

```go
// success
c.JSON(http.StatusOK, respbuilder.Ok(respbuilder.Success, data))

// error
c.JSON(http.StatusBadRequest, respbuilder.Error(respbuilder.ErrValidation, err, "field must be > 0"))
```

Never construct a raw `map[string]interface{}` JSON response inline.

---

## Database

- If a domain introduces its own IDs, generate them as `int64` via **Sonyflake** (`pkg/idgen.NextID()`). Never use `SERIAL`/`IDENTITY`.
- Prefer atomic, guarded `UPDATE ... WHERE <condition>` statements over read-then-write when mutating shared state under concurrency.

---

## Transactions

Service code that needs multiple writes in one atomic unit must:

1. Call `db.BeginTx(ctx, nil)`.
2. Pass the `*sql.Tx` to every repo method in the unit.
3. `tx.Commit()` on success; `tx.Rollback()` on any error (use `defer` with a flag).

---

## Migrations

- Location: `migrations/` directory.
- File naming: `YYYY-MM-DD-HH-mm-SS_{description}.sql` (use `make migration-new name=<description>`).
- Format (rubenv/sql-migrate):

```sql
-- +migrate Up
CREATE TABLE ...;

-- +migrate Down
DROP TABLE ...;
```

- Run migrations: `./server sql-migrate up` (requires `POSTGRES_DSN` env var).

---

## Linting

Run the same linter as CI:

```
make lint
```

Configuration: `.golangci.yml`.  CI uses `golangci-lint v2.12.2` (`golangci/golangci-lint-action@v7`).

---

## Testing

- Package name convention: use `package foo_test` (black-box) for most tests; `package foo` only when testing unexported internals.
- Use `github.com/stretchr/testify/assert` for assertions.
- Prefer table-driven tests with `t.Run`.
- For handler tests: implement the relevant interface with a simple stub struct — no mock generation framework required.
- Integration tests (Postgres) are out of scope for unit test runs; guard with `testing.Short()` skip if added later.

---

## Logging

- Use the global `slog.Default()` logger everywhere. Never create a logger inside a handler or repository.
- Always pass `context.Context` to `slog.*Context` calls so trace IDs propagate.
- Mask sensitive fields (PII, secrets, account numbers) in any log call.

---

## OpenTelemetry Tracing

- Get the span from context: `span := trace.SpanFromContext(ctx)`.
- Create child spans with the span's own TracerProvider (not the global one directly), as shown in `handlersystem/handler.go`.

---

## Key Dependencies (do not substitute without a design discussion)

| Purpose | Package |
|---|---|
| HTTP framework | `github.com/labstack/echo/v4` |
| SQL driver | `github.com/lib/pq` (Postgres) |
| Migrations | `github.com/rubenv/sql-migrate` |
| ID generation | `github.com/sony/sonyflake` |
| UUID validation | `github.com/google/uuid` |
| Validation | `github.com/go-playground/validator/v10` |
| Tracing | `go.opentelemetry.io/otel` |
| Metrics | `github.com/prometheus/client_golang` |
| CLI | `github.com/hashicorp/cli` |
