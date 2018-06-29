package goen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	sqr "gopkg.in/Masterminds/squirrel.v1"
)

func TestPatchCompilerFunc(t *testing.T) {
	assert.Implements(t, (*PatchCompiler)(nil), PatchCompilerFunc(func(opts *CompilerOptions) *SqlizerList {
		return NewSqlizerList()
	}))
}

func TestBulkInsertCompiler(t *testing.T) {
	t.Run("SimpleCase", func(t *testing.T) {
		patches := NewPatchList()
		patches.PushBack(&Patch{
			Kind:      PatchInsert,
			TableName: "testing",
			Columns:   []string{"id", "name"},
			Values:    []interface{}{1, "a"},
		})
		patches.PushBack(&Patch{
			Kind:      PatchInsert,
			TableName: "testing",
			Columns:   []string{"id", "name"},
			Values:    []interface{}{2, "b"},
		})
		patches.PushBack(&Patch{
			Kind:      PatchInsert,
			TableName: "testing",
			Columns:   []string{"id", "name"},
			Values:    []interface{}{3, "c"},
		})
		sqlizers := BulkInsertCompiler.Compile(&CompilerOptions{
			StmtBuilder: sqr.StatementBuilder,
			Patches:     patches,
		})
		if !assert.Equal(t, 1, sqlizers.Len()) {
			return
		}

		sqlizer := sqlizers.Front().GetValue()
		query, args, err := sqlizer.ToSql()
		assert.NoError(t, err)
		assert.Equal(t, `INSERT INTO testing (id,name) VALUES (?,?),(?,?),(?,?)`, query)
		assert.Equal(t, []interface{}{1, "a", 2, "b", 3, "c"}, args)
	})
	t.Run("ComplexCase", func(t *testing.T) {
		patches := NewPatchList()
		patches.PushBack(&Patch{
			Kind:      PatchInsert,
			TableName: "testing",
			Columns:   []string{"id", "name"},
			Values:    []interface{}{1, "a"},
		})
		patches.PushBack(&Patch{
			Kind:      PatchInsert,
			TableName: "testing",
			Columns:   []string{"id", "name"},
			Values:    []interface{}{2, "b"},
		})
		patches.PushBack(&Patch{
			Kind:      PatchUpdate,
			TableName: "testing",
			Columns:   []string{"name"},
			Values:    []interface{}{"c"},
			RowKey: &MapRowKey{
				Table: "testing",
				Key: map[string]interface{}{
					"id": 1,
				},
			},
		})
		patches.PushBack(&Patch{
			Kind:      PatchInsert,
			TableName: "testing",
			Columns:   []string{"id", "name"},
			Values:    []interface{}{3, "c"},
		})
		sqlizers := BulkInsertCompiler.Compile(&CompilerOptions{
			StmtBuilder: sqr.StatementBuilder,
			Patches:     patches,
		})
		if !assert.Equal(t, 3, sqlizers.Len()) {
			t.Log("Got sqlizers:")
			for curr := sqlizers.Front(); curr != nil; curr = curr.Next() {
				query, args, err := curr.GetValue().ToSql()
				if err == nil {
					t.Logf("%q with %v", query, args)
				} else {
					t.Logf("%s", err)
				}
			}
			return
		}
		curr := sqlizers.Front()

		sqlizer := curr.GetValue()
		curr = curr.Next()
		query, args, err := sqlizer.ToSql()
		if assert.NoError(t, err) {
			assert.Equal(t, `INSERT INTO testing (id,name) VALUES (?,?),(?,?)`, query)
			assert.Equal(t, []interface{}{1, "a", 2, "b"}, args)
		}

		sqlizer = curr.GetValue()
		curr = curr.Next()
		query, args, err = sqlizer.ToSql()
		if assert.NoError(t, err) {
			assert.Equal(t, `UPDATE testing SET name = ? WHERE id = ?`, query)
			assert.Equal(t, []interface{}{"c", 1}, args)
		}

		sqlizer = curr.GetValue()
		curr = curr.Next()
		query, args, err = sqlizer.ToSql()
		if assert.NoError(t, err) {
			assert.Equal(t, `INSERT INTO testing (id,name) VALUES (?,?)`, query)
			assert.Equal(t, []interface{}{3, "c"}, args)
		}
	})
}
