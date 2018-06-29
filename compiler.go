package goen

import (
	sqr "gopkg.in/Masterminds/squirrel.v1"
)

var (
	DefaultCompiler PatchCompiler = PatchCompilerFunc(compilePatch)

	BulkInsertCompiler PatchCompiler = PatchCompilerFunc(compileBulkInsert)
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

func compileBulkInsert(opts *CompilerOptions) (sqlizers *SqlizerList) {
	var lastPatch *Patch
	compatLastPatch := func(patch *Patch) bool {
		if lastPatch == nil {
			lastPatch = patch
			return false
		}
		if lastPatch.TableName != patch.TableName {
			return false
		}
		if len(patch.Columns) != len(lastPatch.Columns) {
			return false
		}
		for i := range patch.Columns {
			if patch.Columns[i] != lastPatch.Columns[i] {
				return false
			}
		}
		return true
	}
	fallback := func(patches *PatchList) *SqlizerList {
		opts := &CompilerOptions{
			StmtBuilder: opts.StmtBuilder,
			Patches:     patches,
		}
		return DefaultCompiler.Compile(opts)
	}

	sqlizers = NewSqlizerList()
	var (
		buffer         = NewPatchList()
		bulkInsertStmt sqr.InsertBuilder
		fresh          = true
	)
	for curr := opts.Patches.Front(); curr != nil; curr = curr.Next() {
		patch := curr.GetValue()
		if len(patch.Columns) != len(patch.Values) {
			panic("goen: number of columns and values are mismatched")
		}
		// merge as bulk insert when continuous insert patches
		if patch.Kind == PatchInsert {
			if buffer.Len() > 0 {
				sqlizers.PushBackList(fallback(buffer))
				buffer.Init()
			}
			if compatLastPatch(patch) {
				bulkInsertStmt = bulkInsertStmt.Values(patch.Values...)
			} else {
				if !fresh {
					sqlizers.PushBack(bulkInsertStmt)
				}
				bulkInsertStmt = opts.StmtBuilder.
					Insert(patch.TableName).
					Columns(patch.Columns...).
					Values(patch.Values...)
				fresh = false
			}
		} else {
			if !fresh {
				sqlizers.PushBack(bulkInsertStmt)
				fresh = true
				lastPatch = nil
			}
			buffer.PushBack(patch)
		}
	}
	// last patch is insert, then buffer must be empty
	if buffer.Len() > 0 {
		sqlizers.PushBackList(fallback(buffer))
		buffer.Init()
	}
	if !fresh {
		sqlizers.PushBack(bulkInsertStmt)
	}
	return sqlizers
}
