package goen_test

import (
	"database/sql"
	"io/ioutil"
	"log"
	"testing"

	sqr "github.com/Masterminds/squirrel"
	"github.com/kamichidu/goen"
	_ "github.com/kamichidu/goen/dialect/sqlite3"
	_ "github.com/mattn/go-sqlite3"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

type TestingScanner struct {
	Value uuid.UUID

	ScanCount int
}

func (s *TestingScanner) Scan(src interface{}) error {
	s.ScanCount++
	return s.Value.Scan(src)
}

func TestDBContext(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// create table for testing
	if _, err := db.Exec("create table testing (id integer primary key, name varchar, enabled boolean, uuid blob)"); err != nil {
		panic(err)
	}
	// insert records for testing
	records := [][]interface{}{
		{int64(1), "first", true, uuid.Must(uuid.NewV4())},
		{int64(2), "second", false, uuid.Must(uuid.NewV4())},
		{int64(3), "third", true, uuid.Must(uuid.NewV4())},
	}
	for _, record := range records {
		if _, err := db.Exec("insert into testing (id, name, enabled, uuid) values (?, ?, ?, ?)", record...); err != nil {
			panic(err)
		}
	}

	t.Run("Scan", func(t *testing.T) {
		getRows := func() *sql.Rows {
			rows, err := db.Query("select id, name, enabled, uuid from testing order by id")
			if err != nil {
				panic(err)
			}
			return rows
		}
		dbc := goen.NewDBContext("sqlite3", db)

		t.Run("Slice", func(t *testing.T) {
			rows := getRows()
			defer rows.Close()

			var scannedRecords [][]interface{}
			err := dbc.Scan(rows, &scannedRecords)
			if !assert.NoError(t, err) {
				return
			}
			if !assert.Len(t, scannedRecords, len(records), "match scanned rows") {
				return
			}
			// XXX: sqlite driver returns a column in []uint8 normally
			for i := range records {
				record := records[i]
				scannedRecord := scannedRecords[i]
				// id
				if v, ok := scannedRecord[0].(int64); ok {
					scannedRecord[0] = v
				}
				// name
				if v, ok := scannedRecord[1].([]byte); ok {
					scannedRecord[1] = string(v)
				}
				// enabled
				// sqlite driver returns a bool as a bool type
				// uuid
				if v, ok := scannedRecord[3].([]byte); ok {
					scannedRecord[3] = uuid.FromStringOrNil(string(v))
				}

				assert.Equal(t, record, scannedRecord)
			}
		})
		t.Run("Map", func(t *testing.T) {
			rows := getRows()
			defer rows.Close()

			var scannedRecords []map[string]interface{}
			err := dbc.Scan(rows, &scannedRecords)
			if !assert.NoError(t, err) {
				return
			}
			if !assert.Len(t, scannedRecords, len(records), "match scanned rows") {
				return
			}
			// XXX: sqlite driver returns a column in []uint8 normally
			for i := range records {
				record := records[i]
				scannedRecord := scannedRecords[i]

				if v, ok := scannedRecord["name"].([]byte); ok {
					scannedRecord["name"] = string(v)
				}
				if v, ok := scannedRecord["uuid"].([]byte); ok {
					scannedRecord["uuid"] = uuid.FromStringOrNil(string(v))
				}

				assert.Equal(t, record, []interface{}{
					scannedRecord["id"],
					scannedRecord["name"],
					scannedRecord["enabled"],
					scannedRecord["uuid"],
				})
			}
		})
		t.Run("Struct", func(t *testing.T) {
			type Record struct {
				ID      int64
				Name    string
				Enabled bool
				UUID    uuid.UUID
			}

			rows := getRows()
			defer rows.Close()

			var scannedRecords []Record
			err := dbc.Scan(rows, &scannedRecords)
			if !assert.NoError(t, err) {
				return
			}
			if !assert.Len(t, scannedRecords, len(records), "match scanned rows") {
				return
			}
			for i := range records {
				record := records[i]
				scannedRecord := scannedRecords[i]

				assert.Equal(t, record, []interface{}{
					scannedRecord.ID,
					scannedRecord.Name,
					scannedRecord.Enabled,
					scannedRecord.UUID,
				})
			}
		})
		t.Run("StructPtr", func(t *testing.T) {
			type Record struct {
				ID      int64
				Name    string
				Enabled bool
				UUID    uuid.UUID
			}

			rows := getRows()
			defer rows.Close()

			var scannedRecords []*Record
			err := dbc.Scan(rows, &scannedRecords)
			if !assert.NoError(t, err) {
				return
			}
			if !assert.Len(t, scannedRecords, len(records), "match scanned rows") {
				return
			}
			for i := range records {
				record := records[i]
				scannedRecord := scannedRecords[i]

				assert.Equal(t, record, []interface{}{
					scannedRecord.ID,
					scannedRecord.Name,
					scannedRecord.Enabled,
					scannedRecord.UUID,
				})
			}
		})
		t.Run("Struct with sql.Scanner", func(t *testing.T) {
			type Record struct {
				ID      int64
				Name    string
				Enabled bool
				Scanner TestingScanner `column:"uuid"`
			}

			rows := getRows()
			defer rows.Close()

			var scannedRecords []*Record
			err := dbc.Scan(rows, &scannedRecords)
			if !assert.NoError(t, err) {
				return
			}
			if !assert.Len(t, scannedRecords, len(records), "match scanned rows") {
				return
			}
			for i := range records {
				record := records[i]
				scannedRecord := scannedRecords[i]

				assert.Equal(t, 1, scannedRecord.Scanner.ScanCount, "Scan should call sql.Scanner.Scan")
				assert.Equal(t, record, []interface{}{
					scannedRecord.ID,
					scannedRecord.Name,
					scannedRecord.Enabled,
					scannedRecord.Scanner.Value,
				})
			}
		})
	})
	t.Run("UseTx", func(t *testing.T) {
		tx, err := db.Begin()
		if !assert.NoError(t, err) {
			return
		}
		defer tx.Rollback()

		t.Run("", func(t *testing.T) {
			dbc := goen.NewDBContext("sqlite3", db)
			dbc.Compiler = goen.BulkCompiler
			dbc.Logger = log.New(ioutil.Discard, "", 0)
			txc := dbc.UseTx(tx)
			assert.True(t, dbc.DB == txc.DB,
				"dbc.DB == txc.DB")
			assert.True(t, dbc.Tx == nil && txc.Tx == tx,
				"dbc.Tx == nil && txc.Tx == tx")
			assert.True(t, dbc.Compiler == txc.Compiler,
				"dbc.Compiler == txc.Compiler")
			assert.True(t, dbc.Logger == txc.Logger,
				"dbc.Logger == txc.Logger")
			assert.True(t, dbc.MaxIncludeDepth == txc.MaxIncludeDepth,
				"dbc.MaxIncludeDepth == txc.MaxIncludeDepth")
			assert.True(t, dbc.QueryRunner == db && txc.QueryRunner == tx,
				"dbc.QueryRunner == db && txc.QueryRunner == tx")
		})
		t.Run("", func(t *testing.T) {
			dbc := goen.NewDBContext("sqlite3", db)
			dbc.Compiler = goen.BulkCompiler
			dbc.Logger = log.New(ioutil.Discard, "", 0)
			dbc.QueryRunner = goen.NewStmtCacher(dbc.DB)
			txc := dbc.UseTx(tx)
			assert.True(t, dbc.DB == txc.DB,
				"dbc.DB == txc.DB")
			assert.True(t, dbc.Tx == nil && txc.Tx == tx,
				"dbc.Tx == nil && txc.Tx == tx")
			assert.True(t, dbc.Compiler == txc.Compiler,
				"dbc.Compiler == txc.Compiler")
			assert.True(t, dbc.Logger == txc.Logger,
				"dbc.Logger == txc.Logger")
			assert.True(t, dbc.MaxIncludeDepth == txc.MaxIncludeDepth,
				"dbc.MaxIncludeDepth == txc.MaxIncludeDepth")
			assert.True(t, dbc.QueryRunner != txc.QueryRunner,
				"dbc.QueryRunner != txc.QueryRunner")
			assert.IsType(t, (*goen.StmtCacher)(nil), dbc.QueryRunner,
				"dbc.QueryRunner is *goen.StmtCacher")
			assert.IsType(t, (*goen.StmtCacher)(nil), txc.QueryRunner,
				"txc.QueryRunner is *goen.StmtCacher")
		})
	})
	t.Run("QuerySqlizer", func(t *testing.T) {
		dbc := goen.NewDBContext("sqlite3", db)
		rows, err := dbc.QuerySqlizer(sqr.Expr(`select ? as n`, 99))
		if !assert.NoError(t, err) {
			return
		}
		defer rows.Close()

		var records []struct {
			N int64 `column:"n"`
		}
		err = dbc.Scan(rows, &records)
		if !assert.NoError(t, err) {
			return
		}
		if !assert.Len(t, records, 1) {
			return
		}
		assert.EqualValues(t, 99, records[0].N)
	})
	t.Run("QueryRowSqlizer", func(t *testing.T) {
		dbc := goen.NewDBContext("sqlite3", db)
		row := dbc.QueryRowSqlizer(sqr.Expr(`select ? as n`, 99))

		var n int64
		err := row.Scan(&n)
		if !assert.NoError(t, err) {
			return
		}
		assert.EqualValues(t, 99, n)
	})
}
