package goen

import (
	"container/list"
	"context"
	"database/sql"
	"encoding"
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
		log.Printf("goen: %q with %v", query, args)
	}
	return dbc.stmtCacher.Query(query, args...)
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
		slice := reflect.MakeSlice(rowTyp, len(cols), len(cols))
		scanArgs := make([]interface{}, len(cols))
		for i := range cols {
			scanTyp := dbc.dialect.ScanTypeOf(cols[i])
			scanArg := reflect.New(scanTyp).Elem()
			el := slice.Index(i)
			el.Set(scanArg)
			scanArgs[i] = el.Addr().Interface()
		}
		if err := rows.Scan(scanArgs...); err != nil {
			return reflect.Value{}, err
		}
		return slice, nil
	case reflect.Map:
		scanArgs := make([]interface{}, len(cols))
		for i := range cols {
			scanTyp := dbc.dialect.ScanTypeOf(cols[i])
			scanArg := reflect.New(scanTyp).Elem()
			scanArgs[i] = scanArg.Addr().Interface()
		}
		if err := rows.Scan(scanArgs...); err != nil {
			return reflect.Value{}, err
		}
		m := reflect.MakeMapWithSize(rowTyp, len(cols))
		for i := range cols {
			key := reflect.ValueOf(cols[i].Name())
			value := reflect.ValueOf(scanArgs[i]).Elem()
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
		scanArgs := make([]interface{}, len(cols))
		for i := range cols {
			scanTyp := dbc.dialect.ScanTypeOf(cols[i])
			scanArg := reflect.New(scanTyp).Elem()
			scanArgs[i] = scanArg.Addr().Interface()
		}
		if err := rows.Scan(scanArgs...); err != nil {
			return reflect.Value{}, err
		}
		st := reflect.New(rowTyp).Elem()
		for i := range cols {
			fv := st.FieldByNameFunc(func(name string) bool {
				sf, _ := st.Type().FieldByName(name)
				spec := internal.ColumnSpec(sf.Tag)
				columnName := internal.FirstNotEmpty(spec.Name(), strcase.SnakeCase(name))
				return columnName == cols[i].Name()
			})
			if !fv.IsValid() {
				panic("goen: unknown struct field for column " + cols[i].Name() + " on " + rowTyp.String())
			}
			if err := dbc.scanColumn(fv, reflect.ValueOf(scanArgs[i]).Elem().Interface()); err != nil {
				return reflect.Value{}, err
			}
		}
		return st, nil
	default:
		return reflect.Value{}, fmt.Errorf("goen: unsupported scan type %q", rowTyp)
	}
}

func (dbc *DBContext) scanColumn(v reflect.Value, src interface{}) error {
	if !v.CanSet() {
		panic("goen: v is not settable " + v.String())
	}
	scanner, bu, tu, v := dbc.indirectScanner(v)
	if scanner != nil {
		return scanner.Scan(src)
	}

	switch raw := src.(type) {
	case []byte:
		if bu != nil {
			return bu.UnmarshalBinary(raw)
		}
		if tu != nil {
			return tu.UnmarshalText(raw)
		}
	case string:
		if tu != nil {
			return tu.UnmarshalText([]byte(raw))
		}
	}

	sv := reflect.ValueOf(src)
	if sv.IsValid() && sv.Type().AssignableTo(v.Type()) {
		v.Set(sv)
	} else if sv.Kind() == v.Kind() && sv.Type().ConvertibleTo(v.Type()) {
		v.Set(sv.Convert(v.Type()))
	}
	return nil
}

func (dbc *DBContext) indirectScanner(v reflect.Value) (sql.Scanner, encoding.BinaryUnmarshaler, encoding.TextUnmarshaler, reflect.Value) {
	if v.Kind() != reflect.Ptr && v.Type().Name() != "" && v.CanAddr() {
		v = v.Addr()
	}

	for {
		if v.Kind() != reflect.Ptr {
			break
		}
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		if v.NumMethod() > 0 {
			raw := v.Interface()
			if u, ok := raw.(sql.Scanner); ok {
				return u, nil, nil, v
			}
			if u, ok := raw.(encoding.BinaryUnmarshaler); ok {
				return nil, u, nil, v
			}
			if u, ok := raw.(encoding.TextUnmarshaler); ok {
				return nil, nil, u, v
			}
		}
		v = v.Elem()
	}
	return nil, nil, nil, v
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
			log.Printf("goen: %q with %v", query, args)
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
