// +build postgres

package multiref_test

import (
	"database/sql"
	_ "github.com/kamichidu/goen/dialect/postgres"
	_ "github.com/lib/pq"
	"os"
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
