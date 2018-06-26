package goen

import (
	"container/list"
	"context"
	"database/sql"
	"fmt"
	"log"
	"reflect"

	"github.com/kamichidu/goen/dialect"
	"github.com/kamichidu/goen/internal"
	"github.com/stoewer/go-strcase"
	sqr "gopkg.in/Masterminds/squirrel.v1"
)

type DBContext struct {
	DB *sql.DB

	Compiler PatchCompiler

	Logger Logger

	dialect dialect.Dialect

	debug bool

	patchBuffer *list.List

	stmtBuilder sqr.StatementBuilderType

	stmtCacher sqr.DBProxyContext
}

func NewDBContext(dialectName string, db *sql.DB) *DBContext {
	dialectsMu.RLock()
	dialect, ok := dialects[dialectName]
	dialectsMu.RUnlock()
	if !ok {
		panic(fmt.Sprintf("goen: unknown dialect %q (forgotten import?)", dialectName))
	}
	return &DBContext{
		DB:          db,
		dialect:     dialect,
		patchBuffer: list.New(),
		stmtBuilder: sqr.StatementBuilder.PlaceholderFormat(dialect.PlaceholderFormat()),
		stmtCacher:  sqr.NewStmtCacher(db),
	}
}

func (dbc *DBContext) Dialect() dialect.Dialect {
	return dbc.dialect
}

func (dbc *DBContext) DebugMode(enabled bool) {
	dbc.debug = enabled
}

func (dbc *DBContext) UseTx(tx *sql.Tx) *DBContext {
	// replace db runner and copy state
	clone := &DBContext{
		DB:          dbc.DB,
		dialect:     dbc.dialect,
		debug:       dbc.debug,
		patchBuffer: list.New(),
		stmtBuilder: dbc.stmtBuilder,
		stmtCacher:  sqr.NewStmtCacher(tx),
	}
	clone.patchBuffer.PushBackList(dbc.patchBuffer)
	return clone
}

func (dbc *DBContext) Patch(v *Patch) {
	dbc.patchBuffer.PushBack(v)
}

func (dbc *DBContext) Query(query string, args ...interface{}) (*sql.Rows, error) {
	if dbc.debug {
		dbc.debugPrintf("goen: %q with %v", query, args)
	}
	stmt, err := dbc.stmtCacher.Prepare(query)
	if err != nil {
		return nil, err
	}
	return stmt.Query(args...)
}

func (dbc *DBContext) QueryRow(query string, args ...interface{}) *sql.Row {
	if dbc.debug {
		dbc.debugPrintf("goen: %q with %v", query, args)
	}
	stmt, err := dbc.stmtCacher.Prepare(query)
	if err != nil {
		panic(err)
	}
	return stmt.QueryRow(args...)
}

func (dbc *DBContext) Scan(rows *sql.Rows, v interface{}) error {
	out := reflect.ValueOf(v)
	if out.Kind() != reflect.Ptr || out.IsNil() {
		return fmt.Errorf("goen: Scan only accepts pointer of slice, but got %q", out.Type())
	}
	out = out.Elem()
	rowTyp := out.Type().Elem()
	cols, err := rows.ColumnTypes()
	if err != nil {
		return err
	}
	for rows.Next() {
		rowVal, err := dbc.scanRow(rows, cols, rowTyp)
		if err != nil {
			return err
		}
		out = reflect.Append(out, rowVal)
	}
	reflect.ValueOf(v).Elem().Set(out)
	return nil
}

func (dbc *DBContext) Include(v interface{}, sc *ScopeCache, loader IncludeLoader) error {
	const maxDepth = 10
	recordsList := list.New()
	recordsList.PushBack(v)
	for depth := 0; depth < maxDepth; depth++ {
		nextRecordsList := list.New()
		for records := recordsList.Front(); records != nil; records = records.Next() {
			if err := loader.Load(nextRecordsList, sc, records.Value); err != nil {
				return err
			}
		}
		if nextRecordsList.Len() == 0 {
			return nil
		}
		recordsList = nextRecordsList
	}
	return nil
}

func (dbc *DBContext) scanRow(rows sqr.RowScanner, cols []*sql.ColumnType, rowTyp reflect.Type) (reflect.Value, error) {
	switch rowTyp.Kind() {
	case reflect.Slice:
		dest := make([]interface{}, len(cols))
		args := make([]interface{}, len(cols))
		for i := range cols {
			// if possible to get ScanType, suggest to use its type
			if typ := dbc.dialect.ScanTypeOf(cols[i]); typ != nil {
				dest[i] = reflect.New(typ).Elem()
			}
			args[i] = &dest[i]
		}
		if err := rows.Scan(args...); err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(dest), nil
	case reflect.Map:
		res, err := dbc.scanRow(rows, cols, reflect.SliceOf(reflect.TypeOf((*interface{})(nil)).Elem()))
		if err != nil {
			return res, err
		}
		m := reflect.MakeMapWithSize(rowTyp, len(cols))
		for i := range cols {
			key := reflect.ValueOf(cols[i].Name())
			value := res.Index(i)
			m.SetMapIndex(key, value)
		}
		return m, nil
	case reflect.Ptr:
		res, err := dbc.scanRow(rows, cols, rowTyp.Elem())
		if res.CanAddr() {
			res = res.Addr()
		}
		return res, err
	case reflect.Struct:
		dest := reflect.New(rowTyp).Elem()
		args := make([]interface{}, len(cols))
		for i := range cols {
			rfv := dest.FieldByNameFunc(func(name string) bool {
				field, _ := dest.Type().FieldByName(name)
				spec := internal.ColumnSpec(field.Tag)
				columnName := internal.FirstNotEmpty(spec.Name(), strcase.SnakeCase(name))
				return columnName == cols[i].Name()
			})
			if !rfv.IsValid() {
				panic(fmt.Sprintf("goen: unknown struct field for column %q on %v", cols[i].Name(), rowTyp))
			}
			args[i] = rfv.Addr().Interface()
		}
		if err := rows.Scan(args...); err != nil {
			return reflect.Value{}, err
		}
		return dest, nil
	default:
		return reflect.Value{}, fmt.Errorf("goen: unsupported scan type %q", rowTyp)
	}
}

func (dbc *DBContext) CompilePatch() *list.List {
	patches := *dbc.patchBuffer
	dbc.patchBuffer.Init()
	compiler := dbc.Compiler
	if compiler == nil {
		compiler = DefaultCompiler
	}
	opts := &CompilerOptions{
		StmtBuilder: dbc.stmtBuilder,
		Patches:     &patches,
	}
	return compiler.Compile(opts)
}

func (dbc *DBContext) SaveChanges() error {
	return dbc.SaveChangesContext(context.Background())
}

func (dbc *DBContext) SaveChangesContext(ctx context.Context) error {
	sqlizers := dbc.CompilePatch()
	for curr := sqlizers.Front(); curr != nil; curr = curr.Next() {
		sqlizer := curr.Value.(sqr.Sqlizer)
		query, args, err := sqlizer.ToSql()
		if err != nil {
			return err
		}
		if dbc.debug {
			dbc.debugPrintf("goen: %q with %v", query, args)
		}
		stmt, err := dbc.stmtCacher.PrepareContext(ctx, query)
		if err != nil {
			return err
		}
		if _, err := stmt.ExecContext(ctx, args...); err != nil {
			return err
		}
	}
	return nil
}

func (dbc *DBContext) debugPrint(v ...interface{}) {
	if l, ok := dbc.Logger.(leveledLogger); ok {
		l.Debug(v...)
	} else if dbc.Logger != nil {
		dbc.Logger.Print(v...)
	} else {
		log.Print(v...)
	}
}

func (dbc *DBContext) debugPrintf(format string, args ...interface{}) {
	if l, ok := dbc.Logger.(leveledLogger); ok {
		l.Debugf(format, args...)
	} else if dbc.Logger != nil {
		dbc.Logger.Printf(format, args...)
	} else {
		log.Printf(format, args...)
	}
}
