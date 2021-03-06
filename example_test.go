package goen_test

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/kamichidu/goen"
	_ "github.com/kamichidu/goen/dialect/sqlite3"
	_ "github.com/mattn/go-sqlite3"
)

func Example_bulkOperation() {
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

	// set patch compiler, it compiles patches to bulk
	dbc.Compiler = goen.BulkCompiler

	for i := 0; i < 3; i++ {
		dbc.Patch(goen.InsertPatch(
			"testing",
			[]string{"name"},
			[]interface{}{fmt.Sprintf("name-%d", i)},
		))
	}
	dbc.DebugMode(true)
	dbc.Logger = log.New(os.Stdout, "", 0)
	if err := dbc.SaveChanges(); err != nil {
		panic(err)
	}
	// Output:
	// goen: "INSERT INTO `testing` (`name`) VALUES (?),(?),(?)" with [name-0 name-1 name-2]
}

func Example_queryCount() {
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

	for i := 0; i < 3; i++ {
		dbc.Patch(goen.InsertPatch(
			"testing",
			[]string{"name"},
			[]interface{}{fmt.Sprintf("name-%d", i)},
		))
	}
	if err := dbc.SaveChanges(); err != nil {
		panic(err)
	}

	row := dbc.QueryRow(`select count(*) from testing`)

	var count int64
	if err := row.Scan(&count); err != nil {
		panic(err)
	}
	fmt.Printf("dbc founds %d records\n", count)

	var records []map[string]interface{}
	rows, err := dbc.Query(`select id, name from testing`)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	if err := dbc.Scan(rows, &records); err != nil {
		panic(err)
	}

	fmt.Print("records:\n")
	for _, record := range records {
		fmt.Printf("id=%d name=%q\n", record["id"], record["name"])
	}
	// Output:
	// dbc founds 3 records
	// records:
	// id=1 name="name-0"
	// id=2 name="name-1"
	// id=3 name="name-2"
}

func Example_transaction() {
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
	if txc.Tx != tx {
		panic("UseTx makes clones DBContext and set Tx with given value")
	}
	txc.Patch(&goen.Patch{
		Kind:      goen.PatchInsert,
		TableName: "testing",
		Columns:   []string{"name"},
		Values:    []interface{}{"kamichidu"},
	})
	if err := txc.SaveChanges(); err != nil {
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
		if err == sql.ErrTxDone {
			fmt.Print("txc returns an error sql.ErrTxDone after committed\n")
		}
	}()
	// Output:
	// dbc founds 0 records when not committed yet
	// txc founds 1 records when not committed yet since it's same transaction
	// dbc founds 1 records after committed
	// txc returns an error sql.ErrTxDone after committed
}

func Example_cachingPreparedStatements() {
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

	// goen.StmtCacher caches *sql.Stmt until closed.
	preparedQueryRunner := goen.NewStmtCacher(db)
	defer preparedQueryRunner.Close()

	dbc := goen.NewDBContext("sqlite3", db)
	// set preparedQueryRunner to dbc.QueryRunner.
	dbc.QueryRunner = preparedQueryRunner

	fmt.Printf("cached statements %v\n", preparedQueryRunner.StmtStats().CachedStmts)

	err = goen.TxScope(dbc.DB.Begin())(func(tx *sql.Tx) error {
		// txc also use preparedQueryRunner when dbc uses it.
		txc := dbc.UseTx(tx)
		txc.Patch(&goen.Patch{
			Kind:      goen.PatchInsert,
			TableName: "testing",
			Columns:   []string{"name"},
			Values:    []interface{}{"kamichidu"},
		})
		return txc.SaveChanges()
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("cached statements %v\n", preparedQueryRunner.StmtStats().CachedStmts)

	var count int
	if err := dbc.QueryRow(`select count(*) from testing`).Scan(&count); err != nil {
		panic(err)
	}
	fmt.Printf("%v records found\n", count)
	fmt.Printf("cached statements %v\n", preparedQueryRunner.StmtStats().CachedStmts)
	// Output:
	// cached statements 0
	// cached statements 1
	// 1 records found
	// cached statements 2
}

type exampleHook struct {
	PostInsertBuilderHandler func(squirrel.InsertBuilder) squirrel.Sqlizer
	PostUpdateBuilderHandler func(squirrel.UpdateBuilder) squirrel.Sqlizer
	PostDeleteBuilderHandler func(squirrel.DeleteBuilder) squirrel.Sqlizer
}

func (v *exampleHook) PostInsertBuilder(stmt squirrel.InsertBuilder) squirrel.Sqlizer {
	if v.PostInsertBuilderHandler != nil {
		return v.PostInsertBuilderHandler(stmt)
	} else {
		return stmt
	}
}

func (v *exampleHook) PostUpdateBuilder(stmt squirrel.UpdateBuilder) squirrel.Sqlizer {
	if v.PostUpdateBuilderHandler != nil {
		return v.PostUpdateBuilderHandler(stmt)
	} else {
		return stmt
	}
}

func (v *exampleHook) PostDeleteBuilder(stmt squirrel.DeleteBuilder) squirrel.Sqlizer {
	if v.PostDeleteBuilderHandler != nil {
		return v.PostDeleteBuilderHandler(stmt)
	} else {
		return stmt
	}
}

func ExampleCompilerHook() {
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
	dbc.Compiler = goen.CompilerWithHook(goen.DefaultCompiler, &exampleHook{
		PostInsertBuilderHandler: func(stmt squirrel.InsertBuilder) squirrel.Sqlizer {
			fmt.Println("PostInsertBuilder: hello")
			return stmt
		},
		PostUpdateBuilderHandler: func(stmt squirrel.UpdateBuilder) squirrel.Sqlizer {
			fmt.Println("PostUpdateBuilder: compiler")
			return stmt
		},
		PostDeleteBuilderHandler: func(stmt squirrel.DeleteBuilder) squirrel.Sqlizer {
			fmt.Println("PostDeleteBuilder: hook")
			return stmt
		},
	})
	dbc.Patch(goen.InsertPatch("testing", []string{"name"}, []interface{}{"ExampleCompilerHook"}))
	dbc.Patch(goen.UpdatePatch("testing", []string{"name"}, []interface{}{"ExampleCompilerHook"}, nil))
	dbc.Patch(goen.DeletePatch("testing", nil))
	if err := dbc.SaveChanges(); err != nil {
		panic(err)
	}
	// Output:
	// PostInsertBuilder: hello
	// PostUpdateBuilder: compiler
	// PostDeleteBuilder: hook
}
