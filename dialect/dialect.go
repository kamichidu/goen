package dialect

import (
	"database/sql"
	"reflect"

	sqr "github.com/Masterminds/squirrel"
)

type Dialect interface {
	PlaceholderFormat() sqr.PlaceholderFormat

	Quote(string) string

	ScanTypeOf(*sql.ColumnType) reflect.Type
}
