package sqlite3

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/kamichidu/goen"
	sqr "gopkg.in/Masterminds/squirrel.v1"
)

type dialect struct{}

func (d *dialect) PlaceholderFormat() sqr.PlaceholderFormat {
	return sqr.Question
}

func (d *dialect) Quote(s string) string {
	return "`" + strings.Replace(s, "`", "``", -1) + "`"
}

func (d *dialect) ScanTypeOf(ct *sql.ColumnType) reflect.Type {
	typ := ct.ScanType()
	if typ != nil {
		return typ
	}
	switch dbtype := strings.ToLower(ct.DatabaseTypeName()); {
	case dbtype == "integer":
		return reflect.TypeOf(int(0))
	case strings.HasPrefix(dbtype, "varchar"):
		return reflect.TypeOf("")
	case dbtype == "blob":
		return reflect.SliceOf(reflect.TypeOf(byte(0)))
	default:
		panic(fmt.Sprintf("goen/dialect/sqlite3: unsupported database type name %q", ct.DatabaseTypeName()))
	}
}

func init() {
	goen.Register("sqlite3", &dialect{})
}
