package sqldb

import (
	"context"
	"database/sql"
	"log/slog"
	"time"
)

// Rows is the result set returned by QueryContext.
type Rows interface {
	Next() bool
	Scan(dest ...any) error
	Close() error
	Err() error
}

// Row is the single-row result returned by QueryRowContext.
type Row interface {
	Scan(dest ...any) error
}

// DBTX is satisfied by any connection or transaction adapter, allowing
// repository methods to participate in an external transaction or run standalone.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) Row
}

// Tx is a database transaction that also satisfies DBTX.
type Tx interface {
	DBTX
	Commit() error
	Rollback() error
}

// DB is the top-level database connection required by the service.
type DB interface {
	DBTX
	BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error)
}

// ─── concrete *sql.* wrappers ─────────────────────────────────────────────────

type rows struct{ r *sql.Rows }

func (w *rows) Next() bool             { return w.r.Next() }
func (w *rows) Scan(dest ...any) error { return w.r.Scan(dest...) }
func (w *rows) Close() error           { return w.r.Close() }
func (w *rows) Err() error             { return w.r.Err() }

type row struct{ r *sql.Row }

func (w *row) Scan(dest ...any) error { return w.r.Scan(dest...) }

type sqlTx struct {
	t      *sql.Tx
	logger *slog.Logger
}

func (w *sqlTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	started := time.Now()
	result, err := w.t.ExecContext(ctx, query, args...)
	logQuery(ctx, w.logger, "exec", query, started, err)
	return result, err
}

func (w *sqlTx) QueryContext(ctx context.Context, query string, args ...any) (Rows, error) {
	started := time.Now()
	r, err := w.t.QueryContext(ctx, query, args...)
	logQuery(ctx, w.logger, "query", query, started, err)
	if err != nil {
		return nil, err
	}
	return &rows{r: r}, nil
}

func (w *sqlTx) QueryRowContext(ctx context.Context, query string, args ...any) Row {
	started := time.Now()
	result := &row{r: w.t.QueryRowContext(ctx, query, args...)}
	logQuery(ctx, w.logger, "query_row", query, started, nil)
	return result
}

func (w *sqlTx) Commit() error {
	started := time.Now()
	err := w.t.Commit()
	logOperation(context.Background(), w.logger, "commit", started, err)
	return err
}

func (w *sqlTx) Rollback() error {
	started := time.Now()
	err := w.t.Rollback()
	logOperation(context.Background(), w.logger, "rollback", started, err)
	return err
}

// Adapter wraps *sql.DB to satisfy DB.
type Adapter struct {
	d      *sql.DB
	logger *slog.Logger
}

// New wraps a *sql.DB so it satisfies DB. Passing a logger enables operation
// logging; without one, the adapter emits no logs.
func New(d *sql.DB, logger ...*slog.Logger) *Adapter {
	var operationLogger *slog.Logger
	if len(logger) > 0 {
		operationLogger = logger[0]
	}
	return &Adapter{d: d, logger: operationLogger}
}

func (w *Adapter) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	started := time.Now()
	result, err := w.d.ExecContext(ctx, query, args...)
	logQuery(ctx, w.logger, "exec", query, started, err)
	return result, err
}

func (w *Adapter) QueryContext(ctx context.Context, query string, args ...any) (Rows, error) {
	started := time.Now()
	r, err := w.d.QueryContext(ctx, query, args...)
	logQuery(ctx, w.logger, "query", query, started, err)
	if err != nil {
		return nil, err
	}
	return &rows{r: r}, nil
}

func (w *Adapter) QueryRowContext(ctx context.Context, query string, args ...any) Row {
	started := time.Now()
	result := &row{r: w.d.QueryRowContext(ctx, query, args...)}
	logQuery(ctx, w.logger, "query_row", query, started, nil)
	return result
}

func (w *Adapter) BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error) {
	started := time.Now()
	t, err := w.d.BeginTx(ctx, opts)
	logOperation(ctx, w.logger, "begin_tx", started, err)
	if err != nil {
		return nil, err
	}
	return &sqlTx{t: t, logger: w.logger}, nil
}

func logQuery(ctx context.Context, logger *slog.Logger, operation, query string, started time.Time, err error) {
	if logger == nil {
		return
	}
	attrs := []any{
		slog.String("operation", operation),
		slog.String("query", query),
		slog.Duration("duration", time.Since(started)),
	}
	if err != nil {
		attrs = append(attrs, slog.Any("error", err))
		logger.ErrorContext(ctx, "database operation failed", attrs...)
		return
	}
	logger.DebugContext(ctx, "database operation completed", attrs...)
}

func logOperation(ctx context.Context, logger *slog.Logger, operation string, started time.Time, err error) {
	if logger == nil {
		return
	}
	attrs := []any{
		slog.String("operation", operation),
		slog.Duration("duration", time.Since(started)),
	}
	if err != nil {
		attrs = append(attrs, slog.Any("error", err))
		logger.ErrorContext(ctx, "database operation failed", attrs...)
		return
	}
	logger.DebugContext(ctx, "database operation completed", attrs...)
}

// compile-time interface checks.
var _ Rows = (*rows)(nil)
var _ Row = (*row)(nil)
var _ Tx = (*sqlTx)(nil)
var _ DB = (*Adapter)(nil)
