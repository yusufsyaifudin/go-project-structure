package sqldb

import (
	"context"
	"database/sql"
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

type sqlTx struct{ t *sql.Tx }

func (w *sqlTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return w.t.ExecContext(ctx, query, args...)
}

func (w *sqlTx) QueryContext(ctx context.Context, query string, args ...any) (Rows, error) {
	r, err := w.t.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &rows{r: r}, nil
}

func (w *sqlTx) QueryRowContext(ctx context.Context, query string, args ...any) Row {
	return &row{r: w.t.QueryRowContext(ctx, query, args...)}
}

func (w *sqlTx) Commit() error   { return w.t.Commit() }
func (w *sqlTx) Rollback() error { return w.t.Rollback() }

// Adapter wraps *sql.DB to satisfy DB.
type Adapter struct{ d *sql.DB }

// New wraps a *sql.DB so it satisfies DB.
func New(d *sql.DB) *Adapter { return &Adapter{d: d} }

func (w *Adapter) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return w.d.ExecContext(ctx, query, args...)
}

func (w *Adapter) QueryContext(ctx context.Context, query string, args ...any) (Rows, error) {
	r, err := w.d.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &rows{r: r}, nil
}

func (w *Adapter) QueryRowContext(ctx context.Context, query string, args ...any) Row {
	return &row{r: w.d.QueryRowContext(ctx, query, args...)}
}

func (w *Adapter) BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error) {
	t, err := w.d.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &sqlTx{t: t}, nil
}

// compile-time interface checks.
var _ Rows = (*rows)(nil)
var _ Row = (*row)(nil)
var _ Tx = (*sqlTx)(nil)
var _ DB = (*Adapter)(nil)
