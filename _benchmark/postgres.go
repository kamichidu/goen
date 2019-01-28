// +build postgres

package main

import (
	"database/sql"

	_ "github.com/kamichidu/goen/dialect/postgres"
	_ "github.com/lib/pq"
)

const ddl = `
drop table if exists users;
create table users (
	id uuid primary key,
	name varchar not null,
	created_at timestamp with time zone not null
);
`

func openDB() (string, *sql.DB) {
	const name = "postgres"
	db, err := sql.Open(name, "dbname=testing user=testing password=testing sslmode=disable")
	if err != nil {
		panic(err)
	}
	return name, db
}
