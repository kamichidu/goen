package goen_test

import (
	"database/sql"
	"testing"

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
}
