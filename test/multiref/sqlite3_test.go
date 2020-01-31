// +build !postgres

package multiref_test

import (
	"database/sql"

	_ "github.com/kamichidu/goen/dialect/sqlite3"
	_ "github.com/mattn/go-sqlite3"
)

const ddl = `
drop table if exists parent;
create table parent (
	parent_id uuid primary key,
	group_id uuid not null
);

drop table if exists child;
create table child (
	child_id uuid primary key,
	parent_id uuid not null,
	group_id uuid not null
);
`

const dialectName = "sqlite3"

func prepareDB() (string, *sql.DB) {
	db, err := sql.Open(dialectName, "./sqlite.db")
	if err != nil {
		panic(err)
	}
	if _, err := db.Exec(ddl); err != nil {
		panic(err)
	}
	return dialectName, db
}
