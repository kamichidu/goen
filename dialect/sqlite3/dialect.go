package sqlite3

import (
	"database/sql"
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
	return ct.ScanType()
}

func init() {
	goen.Register("sqlite3", &dialect{})
}
