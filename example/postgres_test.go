// +build postgres

package example

import (
	"database/sql"
	"os"

	_ "github.com/kamichidu/goen/dialect/postgres"
	_ "github.com/lib/pq"
)

const ddl = `
drop table if exists posts;
drop table if exists blogs;

create table blogs (
	blog_id uuid primary key,
	name varchar,
	author varchar(32)
);

create table posts (
	blog_id uuid not null,
	post_id serial not null primary key,
	title varchar not null,
	content varchar not null,
	created_at timestamp with time zone not null,
	updated_at timestamp with time zone not null,
	deleted_at timestamp with time zone,
	foreign key (blog_id) references blogs(blog_id)
);
`

const dialectName = "postgres"

func prepareDB() (string, *sql.DB) {
	connstr := os.Getenv("GOEN_TEST_CONNSTR")
	if connstr == "" {
		connstr = "dbname=testing host=localhost user=testing password=testing sslmode=disable"
	}
	db, err := sql.Open(dialectName, connstr)
	if err != nil {
		panic(err)
	}
	if _, err := db.Exec(ddl); err != nil {
		panic(err)
	}
	return dialectName, db
}
