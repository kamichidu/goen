package goen_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/kamichidu/goen"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ExampleTxScope_funcCall() {
	db, err := sql.Open("sqlite3", "./sqlite.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	dbc := goen.NewDBContext("sqlite3", db)

	err = goen.TxScope(db.Begin())(func(tx *sql.Tx) error {
		txc := dbc.UseTx(tx)

		// manipulate with txc
		txc.QueryRow("select 1")

		// return nil, will be called tx.Commit() by TxScope
		// otherwise, will be called tx.Rollback()
		return nil
	})
	if err != nil {
		panic(err)
	}

	// Output:
}

func ExampleTxScope_do() {
	db, err := sql.Open("sqlite3", "./sqlite.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	dbc := goen.NewDBContext("sqlite3", db)

	err = goen.TxScope(db.Begin()).Do(func(tx *sql.Tx) error {
		txc := dbc.UseTx(tx)

		// manipulate with txc
		txc.QueryRow("select 1")

		// return nil, will be called tx.Commit() by TxScope
		// otherwise, will be called tx.Rollback()
		return nil
	})
	if err != nil {
		panic(err)
	}

	// Output:
}

func TestTxScope(t *testing.T) {
	db, err := sql.Open("sqlite3", "./sqlite.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// close db connection immediately
	db.SetMaxOpenConns(0)
	db.SetMaxIdleConns(0)

	if _, err := db.Exec("drop table if exists testing"); err != nil {
		t.Fatalf("failed to drop schema: %v", err)
	}
	if _, err := db.Exec("create table testing (id integer, msg varchar)"); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	var stat1, stat2 sql.DBStats

	stat1 = db.Stats()
	txScope := goen.TxScope(db.Begin())
	if !assert.NotNil(t, txScope) {
		return
	}
	stat2 = db.Stats()

	assert.Equal(t, stat2.OpenConnections, stat1.OpenConnections+1, "unused TxScope, it leaks a db connection")

	t.Run("Direct", func(t *testing.T) {
		var id int64
		stat1 = db.Stats()
		err := goen.TxScope(db.Begin())(func(tx *sql.Tx) error {
			afe, err := tx.Exec("insert into testing (msg) values (?)", "testing")
			if err != nil {
				return err
			}
			id, err = afe.LastInsertId()
			return err
		})
		stat2 = db.Stats()

		if !assert.NoError(t, err) {
			return
		}
		assert.NotZero(t, id)
		assert.Equal(t, stat2.OpenConnections, stat1.OpenConnections, "use TxScpe once, it automatically commit/rollback given transaction")

		row := db.QueryRow("select msg from testing where rowid = ?", id)
		var msg string
		err = row.Scan(&msg)
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, "testing", msg)
	})
	t.Run("Direct with error", func(t *testing.T) {
		var id int64
		stat1 = db.Stats()
		err := goen.TxScope(db.Begin())(func(tx *sql.Tx) error {
			afe, err := tx.Exec("insert into testing (msg) values (?)", "testing")
			if err != nil {
				return err
			}
			id, err = afe.LastInsertId()
			if err != nil {
				return err
			}
			return errors.New("call rollback")
		})
		stat2 = db.Stats()

		if !assert.EqualError(t, err, "call rollback") {
			return
		}
		assert.NotZero(t, id)
		assert.Equal(t, stat2.OpenConnections, stat1.OpenConnections, "use TxScpe once, it automatically commit/rollback given transaction")

		row := db.QueryRow("select msg from testing where rowid = ?", id)
		var msg string
		err = row.Scan(&msg)
		if !assert.Exactly(t, err, sql.ErrNoRows) {
			return
		}
		assert.Zero(t, msg)
	})
	t.Run("Direct with Begin error", func(t *testing.T) {
		stat1 = db.Stats()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := goen.TxScope(db.BeginTx(ctx, &sql.TxOptions{}))(func(tx *sql.Tx) error {
			panic("never called")
		})
		stat2 = db.Stats()

		assert.Exactly(t, err, context.Canceled)
		assert.Equal(t, stat2.OpenConnections, stat1.OpenConnections, "use TxScpe once, it automatically commit/rollback given transaction")
	})
	t.Run("Do", func(t *testing.T) {
		var id int64
		stat1 = db.Stats()
		err := goen.TxScope(db.Begin())(func(tx *sql.Tx) error {
			afe, err := tx.Exec("insert into testing (msg) values (?)", "testing")
			if err != nil {
				return err
			}
			id, err = afe.LastInsertId()
			return err
		})
		stat2 = db.Stats()

		if !assert.NoError(t, err) {
			return
		}
		assert.NotZero(t, id)
		assert.Equal(t, stat2.OpenConnections, stat1.OpenConnections, "use TxScpe once, it automatically commit/rollback given transaction")

		row := db.QueryRow("select msg from testing where rowid = ?", id)
		var msg string
		err = row.Scan(&msg)
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, "testing", msg)
	})
	t.Run("Do with error", func(t *testing.T) {
		var id int64
		stat1 = db.Stats()
		err := goen.TxScope(db.Begin()).Do(func(tx *sql.Tx) error {
			afe, err := tx.Exec("insert into testing (msg) values (?)", "testing")
			if err != nil {
				return err
			}
			id, err = afe.LastInsertId()
			if err != nil {
				return err
			}
			return errors.New("call rollback")
		})
		stat2 = db.Stats()

		if !assert.EqualError(t, err, "call rollback") {
			return
		}
		assert.NotZero(t, id)
		assert.Equal(t, stat2.OpenConnections, stat1.OpenConnections, "use TxScpe once, it automatically commit/rollback given transaction")

		row := db.QueryRow("select msg from testing where rowid = ?", id)
		var msg string
		err = row.Scan(&msg)
		if !assert.Exactly(t, err, sql.ErrNoRows) {
			return
		}
		assert.Zero(t, msg)
	})
	t.Run("Do with Begin error", func(t *testing.T) {
		stat1 = db.Stats()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := goen.TxScope(db.BeginTx(ctx, &sql.TxOptions{})).Do(func(tx *sql.Tx) error {
			panic("never called")
		})
		stat2 = db.Stats()

		assert.Exactly(t, err, context.Canceled)
		assert.Equal(t, stat2.OpenConnections, stat1.OpenConnections, "use TxScpe once, it automatically commit/rollback given transaction")
	})
	t.Run("panic in tx func will rollback automatically", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		assert.PanicsWithValue(t, "suck", func() {
			goen.TxScope(tx, nil)(func(tx *sql.Tx) error {
				panic("suck")
			})
		})
		assert.EqualError(t, tx.Commit(), sql.ErrTxDone.Error())
	})
}
