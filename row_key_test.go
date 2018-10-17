package goen

import (
	"testing"

	sqr "github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/assert"
)

func TestMapRowKey(t *testing.T) {
	rkey := &MapRowKey{
		Table: "testing",
		Key: map[string]interface{}{
			"a": 2,
			"b": 1,
			"c": 0,
		},
	}
	assert.Equal(t, "testing", rkey.TableName())

	cols, args := rkey.RowKey()
	assert.Equal(t, []string{"a", "b", "c"}, cols)
	assert.Equal(t, []interface{}{2, 1, 0}, args)

	_, _, err := rkey.ToSql()
	if !assert.NoError(t, err) {
		return
	}
	// squirrel.Eq.ToSql() is unstable order
	// then we test by toEq func.
	assert.Equal(t, sqr.Eq{
		"a": 2,
		"b": 1,
		"c": 0,
	}, rkey.toEq())
}
