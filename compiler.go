package goen

import (
	"reflect"

	sqr "github.com/Masterminds/squirrel"
	"github.com/kamichidu/goen/dialect"
)

var (
	DefaultCompiler PatchCompiler = &defaultCompiler{}

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

type compilerOptionsUtils CompilerOptions

func (opts *compilerOptionsUtils) StatementBuilder() sqr.StatementBuilderType {
	v := sqr.StatementBuilder
	if opts.Dialect != nil {
		v = v.PlaceholderFormat(opts.Dialect.PlaceholderFormat())
	}
	return v
}

func (opts *compilerOptionsUtils) Quote(s string) string {
	if opts.Dialect != nil {
		s = opts.Dialect.Quote(s)
	}
	return s
}

func (opts *compilerOptionsUtils) Quotes(l []string) []string {
	out := make([]string, len(l))
	for i := range l {
		out[i] = opts.Quote(l[i])
	}
	return out
}

func (opts *compilerOptionsUtils) RowKeyToSqlizer(rowKey RowKey) sqr.Sqlizer {
	if opts.Dialect != nil {
		return rowKey.ToSqlizerWithDialect(opts.Dialect)
	} else {
		return rowKey
	}
}

type defaultCompiler struct{}

func (*defaultCompiler) Compile(options *CompilerOptions) (sqlizers *SqlizerList) {
	opts := (*compilerOptionsUtils)(options)
	stmtBuilder := opts.StatementBuilder()
	sqlizers = NewSqlizerList()
	for curr := opts.Patches.Front(); curr != nil; curr = curr.Next() {
		patch := curr.GetValue()
		if len(patch.Columns) != len(patch.Values) {
			panic("goen: number of columns and values are mismatched")
		}
		switch patch.Kind {
		case PatchInsert:
			stmt := stmtBuilder.Insert(opts.Quote(patch.TableName)).
				Columns(opts.Quotes(patch.Columns)...).
				Values(patch.Values...)
			sqlizers.PushBack(stmt)
		case PatchUpdate:
			stmt := stmtBuilder.Update(opts.Quote(patch.TableName))
			for i := range patch.Columns {
				stmt = stmt.Set(opts.Quote(patch.Columns[i]), patch.Values[i])
			}
			if patch.RowKey != nil {
				stmt = stmt.Where(opts.RowKeyToSqlizer(patch.RowKey))
			}
			sqlizers.PushBack(stmt)
		case PatchDelete:
			stmt := stmtBuilder.Delete(opts.Quote(patch.TableName))
			if patch.RowKey != nil {
				stmt = stmt.Where(opts.RowKeyToSqlizer(patch.RowKey))
			}
			sqlizers.PushBack(stmt)
		default:
			panic("goen: unable to make sql statement for unknown kind (" + string(patch.Kind) + ")")
		}
	}
	return sqlizers
}

type bulkCompiler struct{}

func (c *bulkCompiler) Compile(options *CompilerOptions) (sqlizers *SqlizerList) {
	opts := (*compilerOptionsUtils)(options)
	stmtBuilder := opts.StatementBuilder()
	sqlizers = NewSqlizerList()
	for curr := opts.Patches.Front(); curr != nil; curr = curr.Next() {
		patch := curr.GetValue()
		if len(patch.Columns) != len(patch.Values) {
			panic("goen: number of columns and values are mismatched")
		}

		switch patch.Kind {
		case PatchInsert:
			stmt := stmtBuilder.Insert(opts.Quote(patch.TableName)).Columns(opts.Quotes(patch.Columns)...).Values(patch.Values...)
			for curr.Next() != nil && c.isCompat(patch, curr.Next().GetValue()) {
				curr = curr.Next()
				stmt = stmt.Values(curr.GetValue().Values...)
			}
			sqlizers.PushBack(stmt)
		case PatchDelete:
			stmt := stmtBuilder.Delete(opts.Quote(patch.TableName))
			cond := sqr.Or{}
			if patch.RowKey != nil {
				cond = append(cond, opts.RowKeyToSqlizer(patch.RowKey))
			}
			for curr.Next() != nil && c.isCompat(patch, curr.Next().GetValue()) {
				curr = curr.Next()
				if np := curr.GetValue(); np.RowKey != nil {
					cond = append(cond, opts.RowKeyToSqlizer(np.RowKey))
				}
			}
			stmt = stmt.Where(cond)
			sqlizers.PushBack(stmt)
		case PatchUpdate:
			stmt := stmtBuilder.Update(opts.Quote(patch.TableName))
			for i := range patch.Columns {
				stmt = stmt.Set(opts.Quote(patch.Columns[i]), patch.Values[i])
			}
			cond := sqr.Or{}
			if patch.RowKey != nil {
				cond = append(cond, opts.RowKeyToSqlizer(patch.RowKey))
			}
			for curr.Next() != nil && c.isCompat(patch, curr.Next().GetValue()) {
				curr = curr.Next()
				if np := curr.GetValue(); np.RowKey != nil {
					cond = append(cond, opts.RowKeyToSqlizer(np.RowKey))
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

func (c *bulkCompiler) isCompat(p1, p2 *Patch) bool {
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
