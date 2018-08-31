package goen

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestQueryRunner(t *testing.T) {
	assert.Implements(t, (*QueryRunner)(nil), (*sql.DB)(nil))
	assert.Implements(t, (*QueryRunner)(nil), (*sql.Tx)(nil))
}

func TestStmtCacher(t *testing.T) {
	assert.Implements(t, (*QueryRunner)(nil), (*StmtCacher)(nil))

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	t.Run("PrepareContext", func(t *testing.T) {
		t.Run("", func(t *testing.T) {
			pqr := NewStmtCacher(db)
			defer pqr.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			stmt1, err := pqr.PrepareContext(ctx, "select 1")
			if !assert.NoError(t, err) {
				return
			}
			stmt2, err := pqr.PrepareContext(ctx, "select 1")
			if !assert.NoError(t, err) {
				return
			}
			assert.True(t, stmt1 == stmt2, "cached stmts are same")
		})
		t.Run("", func(t *testing.T) {
			pqr := NewStmtCacher(db)
			defer pqr.Close()

			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			stmt, err := pqr.PrepareContext(ctx, "select 1")
			assert.EqualError(t, err, context.Canceled.Error())
			assert.Nil(t, stmt, "stmt is nil")
		})
	})
	t.Run("Close", func(t *testing.T) {
		pqr := NewStmtCacher(db)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		stmt, err := pqr.PrepareContext(ctx, "select 1")
		if !assert.NoError(t, err) {
			return
		}
		var v int
		assert.NoError(t, stmt.QueryRowContext(ctx).Scan(&v))
		assert.Equal(t, 1, v)
		if !assert.NoError(t, pqr.Close()) {
			return
		}
		assert.EqualError(t, stmt.QueryRowContext(ctx).Scan(&v), "sql: statement is closed")
	})
}
