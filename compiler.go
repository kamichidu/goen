package goen

import (
	"database/sql/driver"
	"reflect"

	sqr "gopkg.in/Masterminds/squirrel.v1"
)

var (
	DefaultCompiler PatchCompiler = PatchCompilerFunc(compilePatch)

	BulkCompiler PatchCompiler = &bulkCompiler{}
)

type CompilerOptions struct {
	StmtBuilder sqr.StatementBuilderType

	Patches *PatchList
}

type PatchCompiler interface {
	Compile(*CompilerOptions) *SqlizerList
}

type PatchCompilerFunc func(*CompilerOptions) *SqlizerList

func (fn PatchCompilerFunc) Compile(opts *CompilerOptions) (sqlizers *SqlizerList) {
	return fn(opts)
}

func compilePatch(opts *CompilerOptions) (sqlizers *SqlizerList) {
	sqlizers = NewSqlizerList()
	for curr := opts.Patches.Front(); curr != nil; curr = curr.Next() {
		patch := curr.GetValue()
		if len(patch.Columns) != len(patch.Values) {
			panic("goen: number of columns and values are mismatched")
		}
		var sqlizer sqr.Sqlizer
		switch patch.Kind {
		case PatchInsert:
			sqlizer = opts.StmtBuilder.Insert(patch.TableName).
				Columns(patch.Columns...).
				Values(patch.Values...)
		case PatchUpdate:
			stmt := opts.StmtBuilder.Update(patch.TableName)
			for i := range patch.Columns {
				stmt = stmt.Set(patch.Columns[i], patch.Values[i])
			}
			if patch.RowKey != nil {
				stmt = stmt.Where(patch.RowKey)
			}
			sqlizer = stmt
		case PatchDelete:
			stmt := opts.StmtBuilder.Delete(patch.TableName)
			if patch.RowKey != nil {
				stmt = stmt.Where(patch.RowKey)
			}
			sqlizer = stmt
		default:
			panic("goen: unable to make sql statement for unknown kind (" + string(patch.Kind) + ")")
		}
		sqlizers.PushBack(sqlizer)
	}
	return sqlizers
}

type bulkCompiler struct{}

func (compiler *bulkCompiler) Compile(opts *CompilerOptions) (sqlizers *SqlizerList) {
	sqlizers = NewSqlizerList()
	for curr := opts.Patches.Front(); curr != nil; curr = curr.Next() {
		patch := curr.GetValue()
		if len(patch.Columns) != len(patch.Values) {
			panic("goen: number of columns and values are mismatched")
		}

		switch patch.Kind {
		case PatchInsert:
			stmt := opts.StmtBuilder.Insert(patch.TableName).Columns(patch.Columns...).Values(patch.Values...)
			for curr.Next() != nil && compiler.isCompat(patch, curr.Next().GetValue()) {
				curr = curr.Next()
				stmt = stmt.Values(curr.GetValue().Values...)
			}
			sqlizers.PushBack(stmt)
		case PatchDelete:
			stmt := opts.StmtBuilder.Delete(patch.TableName)
			cond := sqr.Or{}
			if patch.RowKey != nil {
				cond = append(cond, patch.RowKey)
			}
			for curr.Next() != nil && compiler.isCompat(patch, curr.Next().GetValue()) {
				curr = curr.Next()
				if np := curr.GetValue(); np.RowKey != nil {
					cond = append(cond, np.RowKey)
				}
			}
			stmt = stmt.Where(cond)
			sqlizers.PushBack(stmt)
		case PatchUpdate:
			stmt := opts.StmtBuilder.Update(patch.TableName)
			for i := range patch.Columns {
				stmt = stmt.Set(patch.Columns[i], patch.Values[i])
			}
			cond := sqr.Or{}
			if patch.RowKey != nil {
				cond = append(cond, patch.RowKey)
			}
			for curr.Next() != nil && compiler.isCompat(patch, curr.Next().GetValue()) {
				curr = curr.Next()
				if np := curr.GetValue(); np.RowKey != nil {
					cond = append(cond, np.RowKey)
				}
			}
			stmt = stmt.Where(cond)
			sqlizers.PushBack(stmt)
		default:
			fallbackOpts := &CompilerOptions{
				StmtBuilder: opts.StmtBuilder,
				Patches:     NewPatchList(),
			}
			fallbackOpts.Patches.PushBack(patch)
			sqlizers.PushBackList(DefaultCompiler.Compile(fallbackOpts))
		}
	}
	return sqlizers
}

func (compiler *bulkCompiler) isCompat(p1, p2 *Patch) bool {
	if p1.Kind != p2.Kind {
		return false
	}
	if p1.TableName != p2.TableName {
		return false
	}
	if len(p1.Columns) != len(p2.Columns) {
		return false
	} else {
		for i := range p1.Columns {
			if p1.Columns[i] != p2.Columns[i] {
				return false
			}
		}
	}
	switch p1.Kind {
	case PatchUpdate:
		if len(p1.Values) != len(p2.Values) {
			return false
		} else {
			for i := range p1.Values {
				if !compiler.isCompatValue(p1.Values[i], p2.Values[i]) {
					return false
				}
			}
		}
	}
	return true
}

func (compiler *bulkCompiler) isCompatValue(v1, v2 interface{}) bool {
	valuer1, ok1 := v1.(driver.Valuer)
	valuer2, ok2 := v2.(driver.Valuer)
	if ok1 != ok2 {
		return false
	} else if ok1 {
		var err error
		v1, err = valuer1.Value()
		if err != nil {
			return false
		}
		v2, err = valuer2.Value()
		if err != nil {
			return false
		}
	}
	return reflect.DeepEqual(v1, v2)
}
