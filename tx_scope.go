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
		defer func() {
			// if invoke rollback twice or already commited, there's no side-effect
			tx.Rollback()
		}()
		if err := fn(tx); err != nil {
			tx.Rollback()
			return err
		} else {
			return tx.Commit()
		}
	})
}
