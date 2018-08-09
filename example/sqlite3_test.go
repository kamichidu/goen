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
	created_at datetime not null,
	updated_at datetime not null,
	deleted_at datetime,
	-- primary key(blog_id, post_id),
	foreign key (blog_id) references blogs(blog_id)
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
