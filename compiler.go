package goen

import (
	sqr "github.com/Masterminds/squirrel"
	"github.com/kamichidu/goen/dialect"
	"reflect"
)

var (
	DefaultCompiler PatchCompiler = PatchCompilerFunc(compilePatch)

	BulkCompiler PatchCompiler = &bulkCompiler{}
)

type CompilerOptions struct {
	Dialect dialect.Dialect

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
	stmtBuilder := sqr.StatementBuilder
	if opts.Dialect != nil {
		stmtBuilder = stmtBuilder.PlaceholderFormat(opts.Dialect.PlaceholderFormat())
	}
	quote := func(s string) string {
		if opts.Dialect != nil {
			s = opts.Dialect.Quote(s)
		}
		return s
	}
	quoteList := func(l []string) []string {
		out := make([]string, len(l))
		for i := range l {
			out[i] = quote(l[i])
		}
		return out
	}
	rowKeyWithDialect := func(v RowKey) sqr.Sqlizer {
		if opts.Dialect != nil {
			return v.ToSqlizerWithDialect(opts.Dialect)
		} else {
			return v
		}
	}
	sqlizers = NewSqlizerList()
	for curr := opts.Patches.Front(); curr != nil; curr = curr.Next() {
		patch := curr.GetValue()
		if len(patch.Columns) != len(patch.Values) {
			panic("goen: number of columns and values are mismatched")
		}
		var sqlizer sqr.Sqlizer
		switch patch.Kind {
		case PatchInsert:
			sqlizer = stmtBuilder.Insert(quote(patch.TableName)).
				Columns(quoteList(patch.Columns)...).
				Values(patch.Values...)
		case PatchUpdate:
			stmt := stmtBuilder.Update(quote(patch.TableName))
			for i := range patch.Columns {
				stmt = stmt.Set(quote(patch.Columns[i]), patch.Values[i])
			}
			if patch.RowKey != nil {
				stmt = stmt.Where(rowKeyWithDialect(patch.RowKey))
			}
			sqlizer = stmt
		case PatchDelete:
			stmt := stmtBuilder.Delete(quote(patch.TableName))
			if patch.RowKey != nil {
				stmt = stmt.Where(rowKeyWithDialect(patch.RowKey))
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
	stmtBuilder := sqr.StatementBuilder
	if opts.Dialect != nil {
		stmtBuilder = stmtBuilder.PlaceholderFormat(opts.Dialect.PlaceholderFormat())
	}
	quote := func(s string) string {
		if opts.Dialect != nil {
			s = opts.Dialect.Quote(s)
		}
		return s
	}
	quoteList := func(l []string) []string {
		out := make([]string, len(l))
		for i := range l {
			out[i] = quote(l[i])
		}
		return out
	}
	rowKeyWithDialect := func(v RowKey) sqr.Sqlizer {
		if opts.Dialect != nil {
			return v.ToSqlizerWithDialect(opts.Dialect)
		} else {
			return v
		}
	}
	sqlizers = NewSqlizerList()
	for curr := opts.Patches.Front(); curr != nil; curr = curr.Next() {
		patch := curr.GetValue()
		if len(patch.Columns) != len(patch.Values) {
			panic("goen: number of columns and values are mismatched")
		}

		switch patch.Kind {
		case PatchInsert:
			stmt := stmtBuilder.Insert(quote(patch.TableName)).Columns(quoteList(patch.Columns)...).Values(patch.Values...)
			for curr.Next() != nil && compiler.isCompat(patch, curr.Next().GetValue()) {
				curr = curr.Next()
				stmt = stmt.Values(curr.GetValue().Values...)
			}
			sqlizers.PushBack(stmt)
		case PatchDelete:
			stmt := stmtBuilder.Delete(quote(patch.TableName))
			cond := sqr.Or{}
			if patch.RowKey != nil {
				cond = append(cond, rowKeyWithDialect(patch.RowKey))
			}
			for curr.Next() != nil && compiler.isCompat(patch, curr.Next().GetValue()) {
				curr = curr.Next()
				if np := curr.GetValue(); np.RowKey != nil {
					cond = append(cond, rowKeyWithDialect(np.RowKey))
				}
			}
			stmt = stmt.Where(cond)
			sqlizers.PushBack(stmt)
		case PatchUpdate:
			stmt := stmtBuilder.Update(quote(patch.TableName))
			for i := range patch.Columns {
				stmt = stmt.Set(quote(patch.Columns[i]), patch.Values[i])
			}
			cond := sqr.Or{}
			if patch.RowKey != nil {
				cond = append(cond, rowKeyWithDialect(patch.RowKey))
			}
			for curr.Next() != nil && compiler.isCompat(patch, curr.Next().GetValue()) {
				curr = curr.Next()
				if np := curr.GetValue(); np.RowKey != nil {
					cond = append(cond, rowKeyWithDialect(np.RowKey))
				}
			}
			stmt = stmt.Where(cond)
			sqlizers.PushBack(stmt)
		default:
			fallbackOpts := &CompilerOptions{
				Dialect: opts.Dialect,
				Patches: NewPatchList(),
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
		// do not use "database/sql/driver".Valuer.
		// it's for converting go type to sql type; type converting.
		// if converts the actual value, maybe illegal implementation.
		if !reflect.DeepEqual(p1.Values, p2.Values) {
			return false
		}
	}
	return true
}
