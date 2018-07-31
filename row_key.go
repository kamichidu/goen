package goen

import (
	sqr "gopkg.in/Masterminds/squirrel.v1"
	"sort"
)

type RowKey interface {
	sqr.Sqlizer

	TableName() string

	RowKey() ([]string, []interface{})
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
	expr := sqr.Eq{}
	for col, val := range key.Key {
		expr[col] = val
	}
	return expr.ToSql()
}
