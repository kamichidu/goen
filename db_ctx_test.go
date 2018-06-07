package goen_test

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/kamichidu/goen"
	_ "github.com/kamichidu/goen/dialect/sqlite3"
	_ "github.com/mattn/go-sqlite3"
)

func ExampleTransaction() {
	db, err := sql.Open("sqlite3", "./sqlite.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	ddl := []string{
		`drop table if exists testing`,
		`create table testing (id integer primary key, name varchar(256))`,
	}
	if _, err := db.Exec(strings.Join(ddl, ";")); err != nil {
		panic(err)
	}

	dbc := goen.NewDBContext("sqlite3", db)

	tx, err := dbc.DB.Begin()
	if err != nil {
		panic(err)
	}
	txc := dbc.UseTx(tx)
	txc.Patch(&goen.Patch{
		Kind:      goen.PatchInsert,
		TableName: "testing",
		Columns:   []string{"name"},
		Values:    []interface{}{"kamichidu"},
	})
	if err := txc.SaveChanges(); err != nil {
		panic(err)
	}

	// Output:
	// dbc founds 0 records when not committed yet
	// txc founds 1 records when not committed yet since it's same transaction
	// dbc founds 1 records after committed
	// txc returns error after committed: "sql: statement is closed"
	func() {
		rows, err := dbc.Query(`select * from testing`)
		if err != nil {
			panic(err)
		}
		defer rows.Close()

		var records []map[string]interface{}
		if err := dbc.Scan(rows, &records); err != nil {
			panic(err)
		}
		fmt.Printf("dbc founds %d records when not committed yet\n", len(records))
	}()
	func() {
		rows, err := txc.Query(`select * from testing`)
		if err != nil {
			panic(err)
		}
		defer rows.Close()

		var records []map[string]interface{}
		if err := txc.Scan(rows, &records); err != nil {
			panic(err)
		}
		fmt.Printf("txc founds %d records when not committed yet since it's same transaction\n", len(records))
	}()

	if err := tx.Commit(); err != nil {
		panic(err)
	}

	func() {
		rows, err := dbc.Query(`select * from testing`)
		if err != nil {
			panic(err)
		}
		defer rows.Close()

		var records []map[string]interface{}
		if err := dbc.Scan(rows, &records); err != nil {
			panic(err)
		}
		fmt.Printf("dbc founds %d records after committed\n", len(records))
	}()
	func() {
		_, err := txc.Query(`select * from testing`)
		fmt.Printf("txc returns error after committed: %q\n", err)
	}()
}
