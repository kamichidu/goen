package dialect

import (
	"database/sql"
	sqr "gopkg.in/Masterminds/squirrel.v1"
	"reflect"
)

type Dialect interface {
	PlaceholderFormat() sqr.PlaceholderFormat

	Quote(string) string

	ScanTypeOf(*sql.ColumnType) reflect.Type
}
