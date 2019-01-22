package goen

import (
	"database/sql"
	"reflect"
	"testing"

	sqr "github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/assert"
)

type testingDialect struct{}

func (*testingDialect) PlaceholderFormat() sqr.PlaceholderFormat {
	return sqr.Question
}

func (*testingDialect) Quote(s string) string {
	return `"` + s + `"`
}

func (*testingDialect) ScanTypeOf(_ *sql.ColumnType) reflect.Type {
	return nil
}

func TestPatchCompilerFunc(t *testing.T) {
	assert.Implements(t, (*PatchCompiler)(nil), PatchCompilerFunc(func(opts *CompilerOptions) *SqlizerList {
		return NewSqlizerList()
	}))
}

func TestDefaultCompiler(t *testing.T) {
	cases := []struct {
		Patches  []*Patch
		Sqlizers []sqr.Sqlizer
	}{
		{
			[]*Patch{
				InsertPatch(
					"testing",
					[]string{"id", "name"},
					[]interface{}{1, "a"},
				),
				InsertPatch(
					"testing",
					[]string{"id", "name"},
					[]interface{}{2, "b"},
				),
				InsertPatch(
					"testing",
					[]string{"id", "name"},
					[]interface{}{3, "c"},
				),
			},
			[]sqr.Sqlizer{
				sqr.Expr(`INSERT INTO "testing" ("id","name") VALUES (?,?)`, 1, "a"),
				sqr.Expr(`INSERT INTO "testing" ("id","name") VALUES (?,?)`, 2, "b"),
				sqr.Expr(`INSERT INTO "testing" ("id","name") VALUES (?,?)`, 3, "c"),
			},
		},
		{
			[]*Patch{
				UpdatePatch("testing", []string{"name"}, []interface{}{"a"}, &MapRowKey{
					Table: "testing",
					Key: map[string]interface{}{
						"id": 1,
					},
				}),
				UpdatePatch("testing", []string{"name"}, []interface{}{"a"}, &MapRowKey{
					Table: "testing",
					Key: map[string]interface{}{
						"id": 2,
					},
				}),
				UpdatePatch("testing", []string{"name"}, []interface{}{"a"}, &MapRowKey{
					Table: "testing",
					Key: map[string]interface{}{
						"id": 3,
					},
				}),
			},
			[]sqr.Sqlizer{
				sqr.Expr(`UPDATE "testing" SET "name" = ? WHERE "id" = ?`, "a", 1),
				sqr.Expr(`UPDATE "testing" SET "name" = ? WHERE "id" = ?`, "a", 2),
				sqr.Expr(`UPDATE "testing" SET "name" = ? WHERE "id" = ?`, "a", 3),
			},
		},
		{
			[]*Patch{
				DeletePatch("testing", &MapRowKey{
					Table: "testing",
					Key: map[string]interface{}{
						"id": 1,
					},
				}),
				DeletePatch("testing", &MapRowKey{
					Table: "testing",
					Key: map[string]interface{}{
						"id": 2,
					},
				}),
				DeletePatch("testing", &MapRowKey{
					Table: "testing",
					Key: map[string]interface{}{
						"id": 3,
					},
				}),
			},
			[]sqr.Sqlizer{
				sqr.Expr(`DELETE FROM "testing" WHERE "id" = ?`, 1),
				sqr.Expr(`DELETE FROM "testing" WHERE "id" = ?`, 2),
				sqr.Expr(`DELETE FROM "testing" WHERE "id" = ?`, 3),
			},
		},
		{
			[]*Patch{
				InsertPatch(
					"testing",
					[]string{"id", "name"},
					[]interface{}{1, "a"},
				),
				UpdatePatch(
					"testing",
					[]string{"name"},
					[]interface{}{"c"},
					&MapRowKey{
						Table: "testing",
						Key: map[string]interface{}{
							"id": 1,
						},
					},
				),
				DeletePatch("testing", &MapRowKey{
					Table: "testing",
					Key: map[string]interface{}{
						"id": 1,
					},
				}),
			},
			[]sqr.Sqlizer{
				sqr.Expr(`INSERT INTO "testing" ("id","name") VALUES (?,?)`, 1, "a"),
				sqr.Expr(`UPDATE "testing" SET "name" = ? WHERE "id" = ?`, "c", 1),
				sqr.Expr(`DELETE FROM "testing" WHERE "id" = ?`, 1),
			},
		},
	}
	for _, c := range cases {
		patches := NewPatchList()
		for _, patch := range c.Patches {
			patches.PushBack(patch)
		}
		sqlizers := DefaultCompiler.Compile(&CompilerOptions{
			Dialect: &testingDialect{},
			Patches: patches,
		})
		if !assert.Equal(t, len(c.Sqlizers), sqlizers.Len()) {
			t.Log("expect sqlizers:")
			for _, sqlizer := range c.Sqlizers {
				query, args, err := sqlizer.ToSql()
				if err != nil {
					t.Log(err)
				} else {
					t.Logf("%q with %v", query, args)
				}
			}
			t.Log("actual sqlizers:")
			for curr := sqlizers.Front(); curr != nil; curr = curr.Next() {
				query, args, err := curr.GetValue().ToSql()
				if err != nil {
					t.Log(err)
				} else {
					t.Logf("%q with %v", query, args)
				}
			}
			continue
		}

		curr := sqlizers.Front()
		for _, sqlizer := range c.Sqlizers {
			expectQuery, expectArgs, err := sqlizer.ToSql()
			if err != nil {
				panic(err)
			}
			query, args, err := curr.GetValue().ToSql()
			if !assert.NoError(t, err) {
				continue
			}
			assert.Equal(t, expectQuery, query)
			assert.Equal(t, expectArgs, args)

			curr = curr.Next()
		}
	}
}

func TestBulkCompiler(t *testing.T) {
	cases := []struct {
		Patches  []*Patch
		Sqlizers []sqr.Sqlizer
	}{
		{
			[]*Patch{
				InsertPatch(
					"testing",
					[]string{"id", "name"},
					[]interface{}{1, "a"},
				),
				InsertPatch(
					"testing",
					[]string{"id", "name"},
					[]interface{}{2, "b"},
				),
				InsertPatch(
					"testing",
					[]string{"id", "name"},
					[]interface{}{3, "c"},
				),
			},
			[]sqr.Sqlizer{
				sqr.Expr(`INSERT INTO "testing" ("id","name") VALUES (?,?),(?,?),(?,?)`, 1, "a", 2, "b", 3, "c"),
			},
		},
		{
			[]*Patch{
				InsertPatch(
					"testing",
					[]string{"id", "name"},
					[]interface{}{1, "a"},
				),
				InsertPatch(
					"testing",
					[]string{"id", "name", "memo"},
					[]interface{}{2, "b", "memo"},
				),
				InsertPatch(
					"testing",
					[]string{"id", "name"},
					[]interface{}{3, "c"},
				),
			},
			[]sqr.Sqlizer{
				sqr.Expr(`INSERT INTO "testing" ("id","name") VALUES (?,?)`, 1, "a"),
				sqr.Expr(`INSERT INTO "testing" ("id","name","memo") VALUES (?,?,?)`, 2, "b", "memo"),
				sqr.Expr(`INSERT INTO "testing" ("id","name") VALUES (?,?)`, 3, "c"),
			},
		},
		{
			[]*Patch{
				UpdatePatch("testing", []string{"name"}, []interface{}{"a"}, &MapRowKey{
					Table: "testing",
					Key: map[string]interface{}{
						"id": 1,
					},
				}),
				UpdatePatch("testing", []string{"name"}, []interface{}{"a"}, &MapRowKey{
					Table: "testing",
					Key: map[string]interface{}{
						"id": 2,
					},
				}),
				UpdatePatch("testing", []string{"name"}, []interface{}{"a"}, &MapRowKey{
					Table: "testing",
					Key: map[string]interface{}{
						"id": 3,
					},
				}),
			},
			[]sqr.Sqlizer{
				sqr.Expr(`UPDATE "testing" SET "name" = ? WHERE ("id" = ? OR "id" = ? OR "id" = ?)`, "a", 1, 2, 3),
			},
		},
		{
			[]*Patch{
				UpdatePatch("testing", []string{"name"}, []interface{}{"a"}, &MapRowKey{
					Table: "testing",
					Key: map[string]interface{}{
						"id": 1,
					},
				}),
				UpdatePatch("testing", []string{"name", "memo"}, []interface{}{"a", "memo"}, &MapRowKey{
					Table: "testing",
					Key: map[string]interface{}{
						"id": 2,
					},
				}),
				UpdatePatch("testing", []string{"name"}, []interface{}{"a"}, &MapRowKey{
					Table: "testing",
					Key: map[string]interface{}{
						"id": 3,
					},
				}),
			},
			[]sqr.Sqlizer{
				sqr.Expr(`UPDATE "testing" SET "name" = ? WHERE ("id" = ?)`, "a", 1),
				sqr.Expr(`UPDATE "testing" SET "name" = ?, "memo" = ? WHERE ("id" = ?)`, "a", "memo", 2),
				sqr.Expr(`UPDATE "testing" SET "name" = ? WHERE ("id" = ?)`, "a", 3),
			},
		},
		{
			[]*Patch{
				DeletePatch("testing", &MapRowKey{
					Table: "testing",
					Key: map[string]interface{}{
						"id": 1,
					},
				}),
				DeletePatch("testing", &MapRowKey{
					Table: "testing",
					Key: map[string]interface{}{
						"id": 2,
					},
				}),
				DeletePatch("testing", &MapRowKey{
					Table: "testing",
					Key: map[string]interface{}{
						"id": 3,
					},
				}),
			},
			[]sqr.Sqlizer{
				sqr.Expr(`DELETE FROM "testing" WHERE ("id" = ? OR "id" = ? OR "id" = ?)`, 1, 2, 3),
			},
		},
		{
			[]*Patch{
				DeletePatch("testing1", &MapRowKey{
					Table: "testing1",
					Key: map[string]interface{}{
						"id": 1,
					},
				}),
				DeletePatch("testing2", &MapRowKey{
					Table: "testing2",
					Key: map[string]interface{}{
						"id": 2,
					},
				}),
				DeletePatch("testing1", &MapRowKey{
					Table: "testing1",
					Key: map[string]interface{}{
						"id": 3,
					},
				}),
			},
			[]sqr.Sqlizer{
				sqr.Expr(`DELETE FROM "testing1" WHERE ("id" = ?)`, 1),
				sqr.Expr(`DELETE FROM "testing2" WHERE ("id" = ?)`, 2),
				sqr.Expr(`DELETE FROM "testing1" WHERE ("id" = ?)`, 3),
			},
		},
		{
			[]*Patch{
				InsertPatch(
					"testing",
					[]string{"id", "name"},
					[]interface{}{1, "a"},
				),
				InsertPatch(
					"testing",
					[]string{"id", "name"},
					[]interface{}{2, "b"},
				),
				UpdatePatch(
					"testing",
					[]string{"name"},
					[]interface{}{"c"},
					&MapRowKey{
						Table: "testing",
						Key: map[string]interface{}{
							"id": 1,
						},
					},
				),
				UpdatePatch(
					"testing",
					[]string{"name"},
					[]interface{}{"c"},
					&MapRowKey{
						Table: "testing",
						Key: map[string]interface{}{
							"id": 2,
						},
					},
				),
				InsertPatch(
					"testing",
					[]string{"id", "name"},
					[]interface{}{3, "c"},
				),
				DeletePatch("testing", &MapRowKey{
					Table: "testing",
					Key: map[string]interface{}{
						"id": 1,
					},
				}),
				DeletePatch("testing", &MapRowKey{
					Table: "testing",
					Key: map[string]interface{}{
						"id": 2,
					},
				}),
			},
			[]sqr.Sqlizer{
				sqr.Expr(`INSERT INTO "testing" ("id","name") VALUES (?,?),(?,?)`, 1, "a", 2, "b"),
				sqr.Expr(`UPDATE "testing" SET "name" = ? WHERE ("id" = ? OR "id" = ?)`, "c", 1, 2),
				sqr.Expr(`INSERT INTO "testing" ("id","name") VALUES (?,?)`, 3, "c"),
				sqr.Expr(`DELETE FROM "testing" WHERE ("id" = ? OR "id" = ?)`, 1, 2),
			},
		},
	}
	for _, c := range cases {
		patches := NewPatchList()
		for _, patch := range c.Patches {
			patches.PushBack(patch)
		}
		sqlizers := BulkCompiler.Compile(&CompilerOptions{
			Dialect: &testingDialect{},
			Patches: patches,
		})
		if !assert.Equal(t, len(c.Sqlizers), sqlizers.Len()) {
			t.Log("expect sqlizers:")
			for _, sqlizer := range c.Sqlizers {
				query, args, err := sqlizer.ToSql()
				if err != nil {
					t.Log(err)
				} else {
					t.Logf("%q with %v", query, args)
				}
			}
			t.Log("actual sqlizers:")
			for curr := sqlizers.Front(); curr != nil; curr = curr.Next() {
				query, args, err := curr.GetValue().ToSql()
				if err != nil {
					t.Log(err)
				} else {
					t.Logf("%q with %v", query, args)
				}
			}
			continue
		}

		curr := sqlizers.Front()
		for _, sqlizer := range c.Sqlizers {
			expectQuery, expectArgs, err := sqlizer.ToSql()
			if err != nil {
				panic(err)
			}
			query, args, err := curr.GetValue().ToSql()
			if !assert.NoError(t, err) {
				continue
			}
			assert.Equal(t, expectQuery, query)
			assert.Equal(t, expectArgs, args)

			curr = curr.Next()
		}
	}
}
