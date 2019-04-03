package goen

import (
	"database/sql"
)

type txScope func(func(*sql.Tx) error) error

func (scope txScope) Do(fn func(*sql.Tx) error) error {
	return scope(fn)
}

func TxScope(tx *sql.Tx, err error) txScope {
	return txScope(func(fn func(tx *sql.Tx) error) error {
		if err != nil {
			return err
		}
		// panic in fn, it will be available for caller
		// tx.Rollback() has no effect for already commited tx; tx.Rollback() always is okay
		defer tx.Rollback() // nolint: errcheck
		if err := fn(tx); err != nil {
			return err
		} else {
			return tx.Commit()
		}
	})
}
