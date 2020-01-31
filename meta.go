package goen

import (
	"encoding"
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/kamichidu/goen/internal"
)

// MetaTable represents a table meta info.
type MetaTable interface {
	// Type gets go struct type associated with this table.
	Type() reflect.Type

	// TableName gets a table name.
	TableName() string

	// PrimaryKey gets primary key meta columns.
	PrimaryKey() []MetaColumn

	// ReferenceKeys gets all reference meta columns by other entities.
	ReferenceKeys() [][]MetaColumn

	// OneToManyReferenceKeys gets all one to many reference meta columns by other entities.
	OneToManyReferenceKeys() [][]MetaColumn

	// ManyToOneReferenceKeys gets all many to one reference meta columns by other entities.
	ManyToOneReferenceKeys() [][]MetaColumn

	// Columns gets all meta columns of this table.
	Columns() []MetaColumn
}

type metaTable struct {
	typ reflect.Type

	tableName string

	primaryKey []MetaColumn

	referenceKeys [][]MetaColumn

	oneToManyReferenceKeys [][]MetaColumn

	manyToOneReferenceKeys [][]MetaColumn

	columns []MetaColumn
}

func (m *metaTable) Type() reflect.Type {
	return m.typ
}

func (m *metaTable) TableName() string {
	return m.tableName
}

func (m *metaTable) PrimaryKey() []MetaColumn {
	return m.primaryKey
}

func (m *metaTable) ReferenceKeys() [][]MetaColumn {
	return m.referenceKeys
}

func (m *metaTable) OneToManyReferenceKeys() [][]MetaColumn {
	return m.oneToManyReferenceKeys
}

func (m *metaTable) ManyToOneReferenceKeys() [][]MetaColumn {
	return m.manyToOneReferenceKeys
}

func (m *metaTable) Columns() []MetaColumn {
	return m.columns
}

var _ MetaTable = (*metaTable)(nil)

// MetaColumn represents a column meta info.
type MetaColumn interface {
	// Field gets go struct field associated with this column.
	Field() reflect.StructField

	// OmitEmpty indicates this column allowing to omit column specifier on a insert statement.
	OmitEmpty() bool

	// PartOfPrimaryKey indicates this column is part of primary key of the table.
	PartOfPrimaryKey() bool

	// ColumnName gets this column name.
	ColumnName() string
}

type metaColumn struct {
	field reflect.StructField

	omitEmpty bool

	partOfPrimaryKey bool

	columnName string
}

func (m *metaColumn) Field() reflect.StructField {
	return m.field
}

func (m *metaColumn) OmitEmpty() bool {
	return m.omitEmpty
}

func (m *metaColumn) PartOfPrimaryKey() bool {
	return m.partOfPrimaryKey
}

func (m *metaColumn) ColumnName() string {
	return m.columnName
}

var _ MetaColumn = (*metaColumn)(nil)

// MetaSchema manages meta schemata computed by struct (tags).
// Provide some utility functions for using with DBContext.
type MetaSchema interface {
	// Register adds given object type to this MetaSchema.
	Register(entity interface{})

	// Compute computes meta schemata for registered entities.
	Compute()

	// LoadOf gets meta schema of table that associated with given entity.
	LoadOf(entity interface{}) MetaTable

	// KeyStringFromRowKey gets identity string for given RowKey.
	KeyStringFromRowKey(RowKey) string

	// PrimaryKeyOf gets RowKey that represents given entity.
	PrimaryKeyOf(entity interface{}) RowKey

	// ReferenceKeysOf gets RowKeys that references to entity by other entities.
	ReferenceKeysOf(entity interface{}) []RowKey

	// OneToManyReferenceKeysOf gets RowKeysone that references to entity by other entities with one to many cardinal.
	OneToManyReferenceKeysOf(entity interface{}) []RowKey

	// ManyToOneReferenceKeysOf gets RowKeysone that references to entity by other entities with many to one cardinal.
	ManyToOneReferenceKeysOf(entity interface{}) []RowKey

	// InsertPatchOf gets a patch that represents insert statement.
	InsertPatchOf(entity interface{}) *Patch

	// UpdatePatchOf gets a patch that represents update statement.
	UpdatePatchOf(entity interface{}) *Patch

	// DeletePatchOf gets a patch that represents delete statement.
	DeletePatchOf(entity interface{}) *Patch
}

type metaSchema struct {
	typlist []reflect.Type

	once sync.Once

	built *sync.Map
}

// NewMetaSchema creates new MetaSchema object.
func NewMetaSchema() MetaSchema {
	return new(metaSchema)
}

func (m *metaSchema) KeyStringFromRowKey(rowKey RowKey) string {
	m.Compute()

	cols, vals := rowKey.RowKey()
	params := make([]string, len(cols))
	for i := range cols {
		var valStr string
		perr := safeDo(func() {
			if m, ok := vals[i].(encoding.TextMarshaler); ok {
				if b, err := m.MarshalText(); err != nil {
					panic(err)
				} else {
					valStr = string(b)
				}
			} else if m, ok := vals[i].(encoding.BinaryMarshaler); ok {
				if b, err := m.MarshalBinary(); err != nil {
					panic(err)
				} else {
					valStr = hex.EncodeToString(b)
				}
			} else {
				valStr = fmt.Sprint(vals[i])
			}
		})
		if perr != nil {
			valStr = perr.Error()
		}
		params[i] = cols[i] + "=" + valStr
	}
	return rowKey.TableName() + ";" + strings.Join(params, ";")
}

func (m *metaSchema) PrimaryKeyOf(entity interface{}) RowKey {
	m.Compute()

	metaT := m.LoadOf(entity)
	rv := reflect.ValueOf(entity)
	rv = reflect.Indirect(rv)
	rowKey := &MapRowKey{}
	rowKey.Table = metaT.TableName()
	rowKey.Key = map[string]interface{}{}
	for _, pk := range metaT.PrimaryKey() {
		rfv := rv.FieldByName(pk.Field().Name)
		rowKey.Key[pk.ColumnName()] = rfv.Interface()
	}
	return rowKey
}

func (m *metaSchema) ReferenceKeysOf(entity interface{}) []RowKey {
	m.Compute()

	metaT := m.LoadOf(entity)
	rv := reflect.ValueOf(entity)
	rv = reflect.Indirect(rv)
	var refes []RowKey
	for _, refeKey := range metaT.ReferenceKeys() {
		refe := &MapRowKey{}
		refe.Table = metaT.TableName()
		refe.Key = map[string]interface{}{}
		for _, col := range refeKey {
			rfv := rv.FieldByName(col.Field().Name)
			refe.Key[col.ColumnName()] = rfv.Interface()
		}
		refes = append(refes, refe)
	}
	return refes
}

func (m *metaSchema) OneToManyReferenceKeysOf(entity interface{}) []RowKey {
	m.Compute()

	metaT := m.LoadOf(entity)
	rv := reflect.ValueOf(entity)
	rv = reflect.Indirect(rv)
	var refes []RowKey
	for _, refeKey := range metaT.OneToManyReferenceKeys() {
		refe := &MapRowKey{}
		refe.Table = metaT.TableName()
		refe.Key = map[string]interface{}{}
		for _, col := range refeKey {
			rfv := rv.FieldByName(col.Field().Name)
			refe.Key[col.ColumnName()] = rfv.Interface()
		}
		refes = append(refes, refe)
	}
	return refes
}

func (m *metaSchema) ManyToOneReferenceKeysOf(entity interface{}) []RowKey {
	m.Compute()

	metaT := m.LoadOf(entity)
	rv := reflect.ValueOf(entity)
	rv = reflect.Indirect(rv)
	var refes []RowKey
	for _, refeKey := range metaT.ManyToOneReferenceKeys() {
		refe := &MapRowKey{}
		refe.Table = metaT.TableName()
		refe.Key = map[string]interface{}{}
		for _, col := range refeKey {
			rfv := rv.FieldByName(col.Field().Name)
			refe.Key[col.ColumnName()] = rfv.Interface()
		}
		refes = append(refes, refe)
	}
	return refes
}

func (m *metaSchema) LoadOf(entity interface{}) MetaTable {
	m.Compute()

	typ := m.typeOf(entity)
	if metaT, ok := m.built.Load(typ); ok {
		return metaT.(MetaTable)
	} else {
		panic("goen: not registered type of " + typ.String())
	}
}

func (m *metaSchema) InsertPatchOf(entity interface{}) *Patch {
	metaT := m.LoadOf(entity)
	var (
		cols = make([]string, 0, len(metaT.Columns()))
		vals = make([]interface{}, 0, len(metaT.Columns()))
	)
	rv := reflect.ValueOf(entity)
	rv = reflect.Indirect(rv)
	for _, metaC := range metaT.Columns() {
		rfv := rv.FieldByName(metaC.Field().Name)
		if !rfv.IsValid() || metaC.OmitEmpty() && isEmptyValue(rfv) {
			continue
		}
		cols = append(cols, metaC.ColumnName())
		vals = append(vals, rfv.Interface())
	}
	return &Patch{
		Kind:      PatchInsert,
		TableName: metaT.TableName(),
		Columns:   cols,
		Values:    vals,
	}
}

func (m *metaSchema) UpdatePatchOf(entity interface{}) *Patch {
	metaT := m.LoadOf(entity)
	var (
		cols = make([]string, 0, len(metaT.Columns()))
		vals = make([]interface{}, 0, len(metaT.Columns()))
	)
	rv := reflect.ValueOf(entity)
	rv = reflect.Indirect(rv)
	for _, metaC := range metaT.Columns() {
		// non-pk columns appears in set clause
		if metaC.PartOfPrimaryKey() {
			continue
		}
		rfv := rv.FieldByName(metaC.Field().Name)
		if !rfv.IsValid() || metaC.OmitEmpty() && isEmptyValue(rfv) {
			continue
		}
		cols = append(cols, metaC.ColumnName())
		vals = append(vals, rfv.Interface())
	}
	return &Patch{
		Kind:      PatchUpdate,
		TableName: metaT.TableName(),
		Columns:   cols,
		Values:    vals,
		// update only given entitty filtered by its primary key
		RowKey: m.PrimaryKeyOf(entity),
	}
}

func (m *metaSchema) DeletePatchOf(entity interface{}) *Patch {
	metaT := m.LoadOf(entity)
	return &Patch{
		Kind:      PatchDelete,
		TableName: metaT.TableName(),
		// delete only given entitty filtered by its primary key
		RowKey: m.PrimaryKeyOf(entity),
	}
}

func (m *metaSchema) typeOf(entity interface{}) reflect.Type {
	typ := reflect.TypeOf(entity)
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		panic("goen: not a struct type")
	}
	return typ
}

func (m *metaSchema) Register(entity interface{}) {
	if m.built != nil {
		panic("goen: already computed meta tables")
	}
	m.typlist = append(m.typlist, m.typeOf(entity))
}

func (m *metaSchema) Compute() {
	m.once.Do(func() {
		m.built = new(sync.Map)
		for _, typ := range m.typlist {
			metaT := m.computeOf(typ)
			m.built.Store(typ, metaT)
		}
	})
}

func elemType(typ reflect.Type) reflect.Type {
	for typ.Kind() == reflect.Ptr || typ.Kind() == reflect.Slice {
		typ = typ.Elem()
	}
	return typ
}

func (m *metaSchema) computeOf(typ reflect.Type) MetaTable {
	if typ.Kind() != reflect.Struct {
		panic("goen: given typ is not a struct")
	}
	strct := internal.NewStructFromReflect(typ)
	tbl := &metaTable{
		typ:       typ,
		tableName: internal.TableName(strct),
	}
	fields := internal.FieldsByFunc(strct.Fields(), internal.IsColumnField)
	for _, field := range fields {
		isPrimaryKey := internal.IsPrimaryKeyField(field)
		col := &metaColumn{
			field:            field.Value().(reflect.StructField),
			partOfPrimaryKey: isPrimaryKey,
			columnName:       internal.ColumnName(field),
			omitEmpty:        internal.OmitEmpty(field),
		}
		if isPrimaryKey {
			tbl.primaryKey = append(tbl.primaryKey, col)
		}
		tbl.columns = append(tbl.columns, col)
	}

	// collect reference keys by other entities
	// it includes typ myself for self-referential
	for _, refeTyp := range m.typlist {
		refeTyp = elemType(refeTyp)
		refeStrct := internal.NewStructFromReflect(refeTyp)
		// expected field types is one of:
		// - []*Typ
		// - []Typ
		// - *Typ
		// - Typ
		refeFields := internal.FieldsByFunc(refeStrct.Fields(), func(refeField internal.StructField) bool {
			if internal.IsIgnoredField(refeField) {
				return false
			} else if !internal.IsForeignKeyField(refeField) {
				return false
			}
			refeFieldTyp := refeField.Type().Value().(reflect.Type)
			refeFieldTyp = elemType(refeFieldTyp)
			return refeFieldTyp == typ
		})
		// referenceKey is typ's table column list
		// so we collect them by foreign key struct tag
		for _, refeField := range refeFields {
			var key []MetaColumn
			refeKey := internal.ReferenceKey(refeField)
			for _, refeColName := range refeKey {
				foreField, ok := internal.FieldByFunc(strct.Fields(), internal.EqColumnName(refeColName))
				if !ok {
					// should panic?
					continue
				}
				key = append(key, &metaColumn{
					field:            foreField.Value().(reflect.StructField),
					partOfPrimaryKey: internal.IsPrimaryKeyField(foreField),
					columnName:       internal.ColumnName(foreField),
					omitEmpty:        internal.OmitEmpty(foreField),
				})
			}
			// now refeTyp == Typ, another entity's field are:
			// []*Typ or []Typ : refeTyp (1) - typ (*)
			// *Typ or Typ : refeTyp (*) - typ (1)
			switch {
			case internal.IsOneToManyField(refeField):
				tbl.oneToManyReferenceKeys = append(tbl.oneToManyReferenceKeys, key)
			case internal.IsManyToOneField(refeField):
				tbl.manyToOneReferenceKeys = append(tbl.manyToOneReferenceKeys, key)
			}
			tbl.referenceKeys = append(tbl.referenceKeys, key)
		}
	}
	return tbl
}

var _ MetaSchema = (*metaSchema)(nil)

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}
