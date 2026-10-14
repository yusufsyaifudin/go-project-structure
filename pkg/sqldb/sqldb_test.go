package sqldb_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yusufsyaifudin/go-project-structure/pkg/sqldb"
)

// ─── minimal in-process SQL driver ───────────────────────────────────────────
//
// Each test opens a *sql.DB with a unique DSN. The DSN is a key into a
// package-level map that holds the desired behavior for that connection.
// This avoids global mutable state while keeping the driver implementation
// entirely self-contained (no new dependencies).

type behavior struct {
	execErr     error
	queryErr    error
	queryCols   []string
	queryVals   []driver.Value // single row; must match len(queryCols)
	beginErr    error
	commitErr   error
	rollbackErr error
}

var (
	driverOnce sync.Once
	bMu        sync.RWMutex
	behaviors  = map[string]*behavior{}
)

func registerBehavior(dsn string, b *behavior) {
	bMu.Lock()
	behaviors[dsn] = b
	bMu.Unlock()
}

func lookupBehavior(dsn string) *behavior {
	bMu.RLock()
	b := behaviors[dsn]
	bMu.RUnlock()
	return b
}

// newTestDB registers a behavior and opens a *sql.DB using the test driver.
func newTestDB(t *testing.T, name string, b *behavior) *sql.DB {
	t.Helper()
	driverOnce.Do(func() { sql.Register("sqldbtest", &minDriver{}) })
	registerBehavior(name, b)
	db, err := sql.Open("sqldbtest", name)
	require.NoError(t, err)
	t.Cleanup(func() {
		bMu.Lock()
		delete(behaviors, name)
		bMu.Unlock()
		_ = db.Close()
	})
	return db
}

// minDriver ───────────────────────────────────────────────────────────────────

type minDriver struct{}

func (d *minDriver) Open(name string) (driver.Conn, error) {
	b := lookupBehavior(name)
	if b == nil {
		return nil, fmt.Errorf("no test behavior for %q", name)
	}
	return &minConn{b: b}, nil
}

// minConn ─────────────────────────────────────────────────────────────────────

type minConn struct{ b *behavior }

func (c *minConn) Prepare(query string) (driver.Stmt, error) {
	return &minStmt{b: c.b}, nil
}

func (c *minConn) Close() error { return nil }

func (c *minConn) Begin() (driver.Tx, error) {
	if c.b.beginErr != nil {
		return nil, c.b.beginErr
	}
	return &minTx{b: c.b}, nil
}

// minStmt ─────────────────────────────────────────────────────────────────────

type minStmt struct{ b *behavior }

func (s *minStmt) Close() error  { return nil }
func (s *minStmt) NumInput() int { return -1 }

func (s *minStmt) Exec(_ []driver.Value) (driver.Result, error) {
	return driver.RowsAffected(1), s.b.execErr
}

func (s *minStmt) Query(_ []driver.Value) (driver.Rows, error) {
	if s.b.queryErr != nil {
		return nil, s.b.queryErr
	}
	return &minRows{b: s.b}, nil
}

// minTx ───────────────────────────────────────────────────────────────────────

type minTx struct{ b *behavior }

func (t *minTx) Commit() error   { return t.b.commitErr }
func (t *minTx) Rollback() error { return t.b.rollbackErr }

// minRows ─────────────────────────────────────────────────────────────────────

type minRows struct {
	b    *behavior
	done bool
}

func (r *minRows) Columns() []string { return r.b.queryCols }
func (r *minRows) Close() error      { return nil }

func (r *minRows) Next(dest []driver.Value) error {
	if r.done || len(r.b.queryVals) == 0 {
		return io.EOF
	}
	r.done = true
	copy(dest, r.b.queryVals)
	return nil
}

// ─── New ──────────────────────────────────────────────────────────────────────

func TestNew(t *testing.T) {
	db := newTestDB(t, "new", &behavior{})
	w := sqldb.New(db)
	assert.NotNil(t, w)
}

// ─── Adapter.ExecContext ──────────────────────────────────────────────────────

func TestAdapter_ExecContext(t *testing.T) {
	db := newTestDB(t, "adapter-exec", &behavior{})
	w := sqldb.New(db)
	result, err := w.ExecContext(context.Background(), "INSERT INTO t VALUES ($1)", 1)
	require.NoError(t, err)
	n, nErr := result.RowsAffected()
	require.NoError(t, nErr)
	assert.EqualValues(t, 1, n)
}

// ─── Adapter.QueryContext ─────────────────────────────────────────────────────

func TestAdapter_QueryContext_Success(t *testing.T) {
	b := &behavior{queryCols: []string{"id"}, queryVals: []driver.Value{int64(42)}}
	db := newTestDB(t, "adapter-query-ok", b)
	w := sqldb.New(db)

	rs, err := w.QueryContext(context.Background(), "SELECT id FROM t")
	require.NoError(t, err)

	require.True(t, rs.Next())
	var id int64
	require.NoError(t, rs.Scan(&id))
	assert.EqualValues(t, 42, id)

	assert.False(t, rs.Next())
	assert.NoError(t, rs.Err())
	assert.NoError(t, rs.Close())
}

func TestAdapter_QueryContext_Error(t *testing.T) {
	wantErr := fmt.Errorf("query failed")
	db := newTestDB(t, "adapter-query-err", &behavior{queryErr: wantErr})
	w := sqldb.New(db)

	rs, err := w.QueryContext(context.Background(), "SELECT id FROM t")
	assert.Nil(t, rs)
	assert.ErrorContains(t, err, wantErr.Error())
}

// ─── Adapter.QueryRowContext ──────────────────────────────────────────────────

func TestAdapter_QueryRowContext(t *testing.T) {
	b := &behavior{queryCols: []string{"val"}, queryVals: []driver.Value{int64(7)}}
	db := newTestDB(t, "adapter-queryrow", b)
	w := sqldb.New(db)

	var val int64
	err := w.QueryRowContext(context.Background(), "SELECT val FROM t").Scan(&val)
	require.NoError(t, err)
	assert.EqualValues(t, 7, val)
}

// ─── Adapter.BeginTx ─────────────────────────────────────────────────────────

func TestAdapter_BeginTx_Success(t *testing.T) {
	db := newTestDB(t, "adapter-begin-ok", &behavior{})
	w := sqldb.New(db)

	txObj, err := w.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	assert.NotNil(t, txObj)
	require.NoError(t, txObj.Rollback())
}

func TestAdapter_BeginTx_Error(t *testing.T) {
	wantErr := fmt.Errorf("begin failed")
	db := newTestDB(t, "adapter-begin-err", &behavior{beginErr: wantErr})
	w := sqldb.New(db)

	txObj, err := w.BeginTx(context.Background(), nil)
	assert.Nil(t, txObj)
	assert.ErrorContains(t, err, wantErr.Error())
}

// ─── Tx.ExecContext ───────────────────────────────────────────────────────────

func TestTx_ExecContext(t *testing.T) {
	db := newTestDB(t, "tx-exec", &behavior{})
	w := sqldb.New(db)
	txObj, err := w.BeginTx(context.Background(), nil)
	require.NoError(t, err)

	result, err := txObj.ExecContext(context.Background(), "INSERT INTO t VALUES ($1)", 1)
	require.NoError(t, err)
	n, _ := result.RowsAffected()
	assert.EqualValues(t, 1, n)
	_ = txObj.Rollback()
}

// ─── Tx.QueryContext ──────────────────────────────────────────────────────────

func TestTx_QueryContext_Success(t *testing.T) {
	b := &behavior{queryCols: []string{"id"}, queryVals: []driver.Value{int64(5)}}
	db := newTestDB(t, "tx-query-ok", b)
	w := sqldb.New(db)
	txObj, err := w.BeginTx(context.Background(), nil)
	require.NoError(t, err)

	rs, err := txObj.QueryContext(context.Background(), "SELECT id FROM t")
	require.NoError(t, err)
	require.True(t, rs.Next())
	var id int64
	require.NoError(t, rs.Scan(&id))
	assert.EqualValues(t, 5, id)
	_ = rs.Close()
	_ = txObj.Rollback()
}

func TestTx_QueryContext_Error(t *testing.T) {
	wantErr := fmt.Errorf("tx query failed")
	db := newTestDB(t, "tx-query-err", &behavior{queryErr: wantErr})
	w := sqldb.New(db)
	txObj, err := w.BeginTx(context.Background(), nil)
	require.NoError(t, err)

	rs, err := txObj.QueryContext(context.Background(), "SELECT id FROM t")
	assert.Nil(t, rs)
	assert.ErrorContains(t, err, wantErr.Error())
}

// ─── Tx.QueryRowContext ───────────────────────────────────────────────────────

func TestTx_QueryRowContext(t *testing.T) {
	b := &behavior{queryCols: []string{"n"}, queryVals: []driver.Value{int64(9)}}
	db := newTestDB(t, "tx-queryrow", b)
	w := sqldb.New(db)
	txObj, err := w.BeginTx(context.Background(), nil)
	require.NoError(t, err)

	var n int64
	err = txObj.QueryRowContext(context.Background(), "SELECT n FROM t").Scan(&n)
	require.NoError(t, err)
	assert.EqualValues(t, 9, n)
	_ = txObj.Rollback()
}

// ─── Tx.Commit / Tx.Rollback ─────────────────────────────────────────────────

func TestTx_Commit(t *testing.T) {
	db := newTestDB(t, "tx-commit", &behavior{})
	w := sqldb.New(db)
	txObj, err := w.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	assert.NoError(t, txObj.Commit())
}

func TestTx_Rollback(t *testing.T) {
	db := newTestDB(t, "tx-rollback", &behavior{})
	w := sqldb.New(db)
	txObj, err := w.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	assert.NoError(t, txObj.Rollback())
}
