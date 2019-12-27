package multiref_test

import (
	"database/sql"
	"os"
	"reflect"
	"testing"

	_ "github.com/kamichidu/goen/dialect/postgres"
	"github.com/kamichidu/goen/test/multiref"
	_ "github.com/lib/pq"
	uuid "github.com/satori/go.uuid"
)

func TestMultiRef(t *testing.T) {
	dbc := multiref.NewDBContext(prepareDB())
	dbc.DebugMode(true)

	parent := &multiref.Parent{
		ParentID: uuid.Must(uuid.NewV4()),
		GroupID:  uuid.Must(uuid.NewV4()),
	}
	dbc.Parent.Insert(parent)

	child := &multiref.Child{
		ChildID:  uuid.Must(uuid.NewV4()),
		ParentID: parent.ParentID,
		GroupID:  parent.GroupID,
	}
	dbc.Child.Insert(child)

	if err := dbc.SaveChanges(); err != nil {
		t.Fatalf("can't save to database: %s", err)
	}

	t.Run("one to many relation", func(t *testing.T) {
		parent, err := dbc.Parent.Select().Include(
			dbc.Parent.IncludeChildren,
		).QueryRow()
		if err != nil {
			t.Errorf("can't select parent")
		}
		if len(parent.Children) != 1 {
			t.Errorf("invalid children length %#v", parent.Children)
		}
		if !reflect.DeepEqual(parent.Children[0], child) {
			t.Errorf("included child %#v and #%v differ", parent.Children, child)
		}
	})

	t.Run("many to one relation", func(t *testing.T) {
		child, err := dbc.Child.Select().Include(
			dbc.Child.IncludeParent,
		).QueryRow()
		if err != nil {
			t.Errorf("can't select child")
		}
		if !reflect.DeepEqual(child.Parent, parent) {
			t.Errorf("included parent %#v and %#v differ", child.Parent, parent)
		}
	})
}

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
