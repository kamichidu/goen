package goen

import (
	"sort"

	sqr "github.com/Masterminds/squirrel"
	"github.com/kamichidu/goen/dialect"
)

type RowKey interface {
	sqr.Sqlizer

	TableName() string

	RowKey() ([]string, []interface{})

	ToSqlizerWithDialect(dialect.Dialect) sqr.Sqlizer
}

type MapRowKey struct {
	Table string

	Key map[string]interface{}
}

func (key *MapRowKey) TableName() string {
	return key.Table
}

func (key *MapRowKey) RowKey() ([]string, []interface{}) {
	cols := make([]string, 0, len(key.Key))
	for col := range key.Key {
		cols = append(cols, col)
	}
	sort.Strings(cols)
	vals := make([]interface{}, len(cols))
	for i := range cols {
		vals[i] = key.Key[cols[i]]
	}
	return cols, vals
}

func (key *MapRowKey) ToSql() (string, []interface{}, error) {
	return key.toEq().ToSql()
}

func (key *MapRowKey) ToSqlizerWithDialect(dialect dialect.Dialect) sqr.Sqlizer {
	eq := key.toEq()
	for col := range eq {
		val := eq[col]
		delete(eq, col)
		eq[dialect.Quote(col)] = val
	}
	return eq
}

// for testing
func (key *MapRowKey) toEq() sqr.Eq {
	// copy to modify column name
	expr := sqr.Eq{}
	for col, val := range key.Key {
		expr[col] = val
	}
	return expr
}
