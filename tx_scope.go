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
		perr := safeDo(func() {
			err = fn(tx)
		})
		if perr != nil {
			tx.Rollback()
			return perr
		} else if err != nil {
			tx.Rollback()
			return err
		} else {
			return tx.Commit()
		}
	})
}
