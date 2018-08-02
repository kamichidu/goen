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

type metaTable struct {
	Typ reflect.Type

	TableName string

	PrimaryKey []*metaColumn

	RefecenceKeys [][]*metaColumn

	Columns []*metaColumn
}

type metaColumn struct {
	Field reflect.StructField

	OmitEmpty bool

	PartOfPrimaryKey bool

	ColumnName string
}

type MetaSchema struct {
	typlist []reflect.Type

	once sync.Once

	built *sync.Map
}

func (m *MetaSchema) KeyStringFromRowKey(rowKey RowKey) string {
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

func (m *MetaSchema) PrimaryKeyOf(entity interface{}) RowKey {
	m.Compute()

	metaT := m.LoadOf(entity)
	rv := reflect.ValueOf(entity)
	rv = reflect.Indirect(rv)
	rowKey := &MapRowKey{}
	rowKey.Table = metaT.TableName
	rowKey.Key = map[string]interface{}{}
	for _, pk := range metaT.PrimaryKey {
		rfv := rv.FieldByName(pk.Field.Name)
		rowKey.Key[pk.ColumnName] = rfv.Interface()
	}
	return rowKey
}

func (m *MetaSchema) ReferenceKeysOf(entity interface{}) []RowKey {
	m.Compute()

	metaT := m.LoadOf(entity)
	rv := reflect.ValueOf(entity)
	rv = reflect.Indirect(rv)
	var refes []RowKey
	for _, refeKey := range metaT.RefecenceKeys {
		refe := &MapRowKey{}
		refe.Table = metaT.TableName
		refe.Key = map[string]interface{}{}
		for _, col := range refeKey {
			rfv := rv.FieldByName(col.Field.Name)
			refe.Key[col.ColumnName] = rfv.Interface()
		}
		refes = append(refes, refe)
	}
	return refes
}

func (m *MetaSchema) LoadOf(entity interface{}) *metaTable {
	m.Compute()

	typ := m.typeOf(entity)
	if metaT, ok := m.built.Load(typ); ok {
		return metaT.(*metaTable)
	} else {
		panic("goen: not registered type of " + typ.String())
	}
}

func (m *MetaSchema) InsertPatchOf(entity interface{}) *Patch {
	metaT := m.LoadOf(entity)
	var (
		cols = make([]string, 0, len(metaT.Columns))
		vals = make([]interface{}, 0, len(metaT.Columns))
	)
	rv := reflect.ValueOf(entity)
	rv = reflect.Indirect(rv)
	for _, metaC := range metaT.Columns {
		rfv := rv.FieldByName(metaC.Field.Name)
		if !rfv.IsValid() || metaC.OmitEmpty && isEmptyValue(rfv) {
			continue
		}
		cols = append(cols, metaC.ColumnName)
		vals = append(vals, rfv.Interface())
	}
	return &Patch{
		Kind:      PatchInsert,
		TableName: metaT.TableName,
		Columns:   cols,
		Values:    vals,
	}
}

func (m *MetaSchema) UpdatePatchOf(entity interface{}) *Patch {
	metaT := m.LoadOf(entity)
	var (
		cols = make([]string, 0, len(metaT.Columns))
		vals = make([]interface{}, 0, len(metaT.Columns))
	)
	rv := reflect.ValueOf(entity)
	rv = reflect.Indirect(rv)
	for _, metaC := range metaT.Columns {
		// non-pk columns appears in set clause
		if metaC.PartOfPrimaryKey {
			continue
		}
		rfv := rv.FieldByName(metaC.Field.Name)
		if !rfv.IsValid() || metaC.OmitEmpty && isEmptyValue(rfv) {
			continue
		}
		cols = append(cols, metaC.ColumnName)
		vals = append(vals, rfv.Interface())
	}
	return &Patch{
		Kind:      PatchUpdate,
		TableName: metaT.TableName,
		Columns:   cols,
		Values:    vals,
		// update only given entitty filtered by its primary key
		RowKey: m.PrimaryKeyOf(entity),
	}
}

func (m *MetaSchema) DeletePatchOf(entity interface{}) *Patch {
	metaT := m.LoadOf(entity)
	return &Patch{
		Kind:      PatchDelete,
		TableName: metaT.TableName,
		// delete only given entitty filtered by its primary key
		RowKey: m.PrimaryKeyOf(entity),
	}
}

func (m *MetaSchema) typeOf(entity interface{}) reflect.Type {
	typ := reflect.TypeOf(entity)
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		panic("goen: not a struct type")
	}
	return typ
}

func (m *MetaSchema) Register(entity interface{}) {
	if m.built != nil {
		panic("goen: already computed meta tables")
	}
	m.typlist = append(m.typlist, m.typeOf(entity))
}

func (m *MetaSchema) Compute() {
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

func (m *MetaSchema) computeOf(typ reflect.Type) *metaTable {
	if typ.Kind() != reflect.Struct {
		panic("goen: given typ is not a struct")
	}
	strct := internal.NewStructFromReflect(typ)
	tbl := &metaTable{
		Typ:       typ,
		TableName: internal.TableName(strct),
	}
	fields := internal.FieldsByFunc(strct.Fields(), internal.IsColumnField)
	for _, field := range fields {
		isPrimaryKey := internal.IsPrimaryKeyField(field)
		col := &metaColumn{
			Field:            field.Value().(reflect.StructField),
			PartOfPrimaryKey: isPrimaryKey,
			ColumnName:       internal.ColumnName(field),
			OmitEmpty:        internal.OmitEmpty(field),
		}
		if isPrimaryKey {
			tbl.PrimaryKey = append(tbl.PrimaryKey, col)
		}
		tbl.Columns = append(tbl.Columns, col)
	}

	// collect reference keys by other entities
	// it includes typ myself for self-referential
	for _, refeTyp := range m.typlist {
		refeTyp = elemType(refeTyp)
		refeStrct := internal.NewStructFromReflect(refeTyp)
		// expected field types is one of:
		// - []*RefeTyp
		// - []RefeTyp
		// - *RefeTyp
		// - RefeTyp
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
			var key []*metaColumn
			refeKey := internal.ReferenceKey(refeField)
			for _, refeColName := range refeKey {
				foreField, ok := internal.FieldByFunc(strct.Fields(), internal.EqColumnName(refeColName))
				if !ok {
					// should panic?
					continue
				}
				key = append(key, &metaColumn{
					Field:            foreField.Value().(reflect.StructField),
					PartOfPrimaryKey: internal.IsPrimaryKeyField(foreField),
					ColumnName:       internal.ColumnName(foreField),
					OmitEmpty:        internal.OmitEmpty(foreField),
				})
			}
			tbl.RefecenceKeys = append(tbl.RefecenceKeys, key)
		}
	}
	return tbl
}

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
