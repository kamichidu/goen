package goen

import (
	"context"
	"database/sql"
	sqr "github.com/Masterminds/squirrel"
	"sync"
)

// QueryRunner ...
type QueryRunner interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)

	QueryRowContext(context.Context, string, ...interface{}) *sql.Row

	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
}

// StmtCacher implements QueryRunner.
// It caches prepared statements for executing or querying.
type StmtCacher struct {
	prep sqr.PreparerContext

	cache map[string]*sql.Stmt

	mu sync.Mutex
}

// NewStmtCacher returns new StmtCacher with given prep.
func NewStmtCacher(prep sqr.PreparerContext) *StmtCacher {
	return &StmtCacher{
		prep:  prep,
		cache: map[string]*sql.Stmt{},
	}
}

// StmtStats holds some statistics values of StmtCacher.
type StmtStats struct {
	CachedStmts int
}

// StmtStats returns statistics values.
func (pqr *StmtCacher) StmtStats() StmtStats {
	pqr.mu.Lock()
	defer pqr.mu.Unlock()

	var stats StmtStats
	stats.CachedStmts = len(pqr.cache)
	return stats
}

// Prepare is only for implements squirrel.PreparerContext.
// DO NOT USE THIS.
func (pqr *StmtCacher) Prepare(query string) (*sql.Stmt, error) {
	return pqr.PrepareContext(context.Background(), query)
}

// PrepareContext returns new or cached prepared statement.
func (pqr *StmtCacher) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	pqr.mu.Lock()
	defer pqr.mu.Unlock()

	stmt, ok := pqr.cache[query]
	if ok {
		return stmt, nil
	}
	stmt, err := pqr.prep.PrepareContext(ctx, query)
	if err == nil {
		pqr.cache[query] = stmt
	}
	return stmt, err
}

// Close closes and removes all cached prepared statements.
func (pqr *StmtCacher) Close() error {
	pqr.mu.Lock()
	defer pqr.mu.Unlock()

	for _, stmt := range pqr.cache {
		stmt.Close()
	}
	pqr.cache = map[string]*sql.Stmt{}
	return nil
}

// ExecContext executes given query with args via prepared statement.
func (pqr *StmtCacher) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	stmt, err := pqr.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	return stmt.ExecContext(ctx, args...)
}

// QueryRowContext queries a row by given query with args via prepared statement.
func (pqr *StmtCacher) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	stmt, err := pqr.PrepareContext(ctx, query)
	if err != nil {
		panic(err)
	}
	return stmt.QueryRowContext(ctx, args...)
}

// QueryContext queries by given query with args via prepared statement.
func (pqr *StmtCacher) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	stmt, err := pqr.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	return stmt.QueryContext(ctx, args...)
}

type txPreparer struct {
	tx *sql.Tx

	sqr.PreparerContext
}

func (prep *txPreparer) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	stmt, err := prep.PreparerContext.PrepareContext(ctx, query)
	if err != nil {
		return stmt, err
	}
	return prep.tx.StmtContext(ctx, stmt), nil
}
