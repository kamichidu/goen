// +build !postgres

package example

import (
	"database/sql"

	_ "github.com/kamichidu/goen/dialect/sqlite3"
	_ "github.com/mattn/go-sqlite3"
)

const ddl = `
drop table if exists posts;
drop table if exists blogs;

create table blogs (
	blog_id blob primary key,
	name varchar,
	author varchar(32)
);

create table posts (
	blog_id blob not null,
	post_id integer not null primary key,
	title varchar not null,
	content varchar not null,
	"order" integer,
	created_at datetime not null,
	updated_at datetime not null,
	deleted_at datetime,
	foreign key (blog_id) references blogs(blog_id)
);
`

// sqlite3 does not support "drop constraint"
// so re-create tables without foreign keys
const ddlNoForeignKeys = `
drop table if exists posts;
drop table if exists blogs;

create table blogs (
	blog_id blob primary key,
	name varchar,
	author varchar(32)
);

create table posts (
	blog_id blob not null,
	post_id integer not null primary key,
	title varchar not null,
	content varchar not null,
	"order" integer,
	created_at datetime not null,
	updated_at datetime not null,
	deleted_at datetime
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

func dropForeignKeys(db *sql.DB) {
	_, err := db.Exec(ddlNoForeignKeys)
	if err != nil {
		panic(err)
	}
}
