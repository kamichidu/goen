// +build !postgres

package main

import (
	"database/sql"

	_ "github.com/kamichidu/goen/dialect/sqlite3"
	_ "github.com/mattn/go-sqlite3"
)

const ddl = `
drop table if exists users;
create table users (
	id blob primary key,
	name varchar not null,
	created_at datetime not null
);
`

func openDB() (string, *sql.DB) {
	const name = "sqlite3"
	db, err := sql.Open(name, "./sqlite3.db")
	if err != nil {
		panic(err)
	}
	return name, db
}
