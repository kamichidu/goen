package goen

import (
	"container/list"
	"context"
	"database/sql"
	"fmt"
	"log"
	"reflect"

	sqr "github.com/Masterminds/squirrel"
	"github.com/kamichidu/goen/dialect"
	"github.com/kamichidu/goen/internal"
)

// DBContext holds *sql.DB (and *sql.Tx) with contextual values.
// goen is intended to work with *sql.DB (and *sql.Tx) via DBContext.
type DBContext struct {
	// This field is readonly.
	// Sets via NewDBContext().
	DB *sql.DB

	// This field is readonly.
	// Sets via UseTx(); or nil.
	Tx *sql.Tx

	// The patch compiler.
	// This is used when need to compile patch to query.
	Compiler PatchCompiler

	// The logger for debugging.
	// This is used when DebugMode(true).
	Logger Logger

	// The depth limit to avoid circular Include work.
	// When this value is positive, Include works N-times.
	// When this value is negative, Include works unlimited.
	// When this value is 0, Include never works.
	// Default is 10.
	MaxIncludeDepth int

	// The runner for each query.
	// This field is indented to hold one of *sql.DB, *sql.Tx or *StmtCacher.
	QueryRunner QueryRunner

	dialect dialect.Dialect

	debug bool

	patchBuffer *PatchList

	stmtBuilder sqr.StatementBuilderType
}

// NewDBContext creates DBContext with given dialectName and db.
func NewDBContext(dialectName string, db *sql.DB) *DBContext {
	dialectsMu.RLock()
	dialect, ok := dialects[dialectName]
	dialectsMu.RUnlock()
	if !ok {
		panic(fmt.Sprintf("goen: unknown dialect %q (forgotten import?)", dialectName))
	}
	return &DBContext{
		DB:              db,
		MaxIncludeDepth: 10,
		QueryRunner:     db,
		dialect:         dialect,
		patchBuffer:     NewPatchList(),
		stmtBuilder:     sqr.StatementBuilder.PlaceholderFormat(dialect.PlaceholderFormat()),
	}
}

// Dialect returns dialect.Dialect holds by this context.
func (dbc *DBContext) Dialect() dialect.Dialect {
	return dbc.dialect
}

// DebugMode sets debug flag.
func (dbc *DBContext) DebugMode(enabled bool) {
	dbc.debug = enabled
}

// UseTx returns clone of current DBContext with given *sql.Tx.
func (dbc *DBContext) UseTx(tx *sql.Tx) *DBContext {
	// replace db runner and copy state
	var clone DBContext
	clone = *dbc
	clone.Tx = tx
	if prep, ok := clone.QueryRunner.(*StmtCacher); ok {
		txPrep := &txPreparer{tx: tx, PreparerContext: prep}
		clone.QueryRunner = NewStmtCacher(txPrep)
	} else {
		clone.QueryRunner = tx
	}
	clone.patchBuffer = NewPatchList()
	clone.patchBuffer.PushBackList(dbc.patchBuffer)
	return &clone
}

// Patch adds raw patch into the buffer; without executing a query.
func (dbc *DBContext) Patch(v *Patch) {
	dbc.patchBuffer.PushBack(v)
}

func (dbc *DBContext) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return dbc.QueryContext(context.Background(), query, args...)
}

func (dbc *DBContext) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if dbc.debug {
		dbc.debugPrintf("goen: %q with %v", query, args)
	}
	return dbc.QueryRunner.QueryContext(ctx, query, args...)
}

func (dbc *DBContext) QueryRow(query string, args ...interface{}) *sql.Row {
	return dbc.QueryRowContext(context.Background(), query, args...)
}

func (dbc *DBContext) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if dbc.debug {
		dbc.debugPrintf("goen: %q with %v", query, args)
	}
	return dbc.QueryRunner.QueryRowContext(ctx, query, args...)
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
	return dbc.IncludeContext(context.Background(), v, sc, loader)
}

func (dbc *DBContext) IncludeContext(ctx context.Context, v interface{}, sc *ScopeCache, loader IncludeLoader) error {
	recordsList := list.New()
	recordsList.PushBack(v)
	for depth := 0; depth < dbc.MaxIncludeDepth; depth++ {
		nextRecordsList := list.New()
		for records := recordsList.Front(); records != nil; records = records.Next() {
			if err := loader.Load(ctx, (*IncludeBuffer)(nextRecordsList), sc, records.Value); err != nil {
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

func (dbc *DBContext) CompilePatch() *SqlizerList {
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
		sqlizer := curr.GetValue()
		query, args, err := sqlizer.ToSql()
		if err != nil {
			return err
		}
		if dbc.debug {
			dbc.debugPrintf("goen: %q with %v", query, args)
		}
		if _, err := dbc.QueryRunner.ExecContext(ctx, query, args...); err != nil {
			return err
		}
	}
	return nil
}

// scanRow scans a row as given rowTyp.
func (dbc *DBContext) scanRow(rows sqr.RowScanner, cols []*sql.ColumnType, rowTyp reflect.Type) (reflect.Value, error) {
	switch rowTyp.Kind() {
	case reflect.Slice:
		return dbc.scanRowAsSlice(rows, cols, rowTyp)
	case reflect.Map:
		return dbc.scanRowAsMap(rows, cols, rowTyp)
	case reflect.Ptr:
		res, err := dbc.scanRow(rows, cols, rowTyp.Elem())
		if res.CanAddr() {
			res = res.Addr()
		}
		return res, err
	case reflect.Struct:
		return dbc.scanRowAsStruct(rows, cols, rowTyp)
	default:
		return reflect.Value{}, fmt.Errorf("goen: unsupported scan type %q", rowTyp)
	}
}

// scanRowAsSlice scans a row as slice.
func (dbc *DBContext) scanRowAsSlice(rows sqr.RowScanner, cols []*sql.ColumnType, rowTyp reflect.Type) (reflect.Value, error) {
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
}

// scanRowAsMap scans a row as map.
func (dbc *DBContext) scanRowAsMap(rows sqr.RowScanner, cols []*sql.ColumnType, rowTyp reflect.Type) (reflect.Value, error) {
	rvals, err := dbc.scanRowAsSlice(rows, cols, reflect.TypeOf([]interface{}{}))
	if err != nil {
		return reflect.Value{}, err
	}
	dest := map[string]interface{}{}
	for i := range cols {
		dest[cols[i].Name()] = rvals.Index(i).Interface()
	}
	return reflect.ValueOf(dest), nil
}

// scanRowAsStruct scans a row as struct.
func (dbc *DBContext) scanRowAsStruct(rows sqr.RowScanner, cols []*sql.ColumnType, rowTyp reflect.Type) (reflect.Value, error) {
	dest := reflect.New(rowTyp).Elem()
	strct := internal.NewStructFromReflect(rowTyp)
	fields := internal.FieldsByFunc(strct.Fields(), internal.IsColumnField)
	args := make([]interface{}, len(cols))
	for i := range cols {
		var rfv reflect.Value
		if field, ok := internal.FieldByFunc(fields, internal.EqColumnName(cols[i].Name())); ok {
			rfv = dest.FieldByName(field.Name())
		}
		if !rfv.IsValid() {
			panic(fmt.Sprintf("goen: unknown struct field for column %q on %v", cols[i].Name(), rowTyp))
		}
		args[i] = rfv.Addr().Interface()
	}
	if err := rows.Scan(args...); err != nil {
		return reflect.Value{}, err
	}
	return dest, nil
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
