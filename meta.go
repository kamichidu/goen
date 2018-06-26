package goen

import (
	"encoding"
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/kamichidu/goen/internal"
	"github.com/stoewer/go-strcase"
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

func (m *MetaSchema) KeyStringFromRowKey(rowKey RowKey) (string, error) {
	m.Compute()

	cols, vals := rowKey.RowKey()
	params := make([]string, len(cols))
	for i := range cols {
		if m, ok := vals[i].(encoding.TextMarshaler); ok {
			b, err := m.MarshalText()
			if err != nil {
				return "", err
			}
			params[i] = cols[i] + "=" + string(b)
		} else if m, ok := vals[i].(encoding.BinaryMarshaler); ok {
			b, err := m.MarshalBinary()
			if err != nil {
				return "", err
			}
			params[i] = cols[i] + "=" + hex.EncodeToString(b)
		} else {
			params[i] = cols[i] + "=" + fmt.Sprint(vals[i])
		}
	}
	return rowKey.TableName() + ";" + strings.Join(params, ";"), nil
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

func (m *MetaSchema) RowKeysOf(entity interface{}) (primary RowKey, refes []RowKey) {
	m.Compute()

	primary = m.PrimaryKeyOf(entity)

	metaT := m.LoadOf(entity)
	rv := reflect.ValueOf(entity)
	rv = reflect.Indirect(rv)
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
	return primary, refes
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

func (m *MetaSchema) computeOf(typ reflect.Type) *metaTable {
	fields := m.structFieldsOf(typ)
	metaT := &metaTable{
		Typ: typ,
	}
	for _, field := range fields {
		if _, ok := internal.FirstLookup(field.Tag, internal.TagTable, internal.TagView); ok {
			spec := internal.TableSpec(field.Tag)
			metaT.TableName = spec.Name()
		} else if _, ok := field.Tag.Lookup(internal.TagIgnore); ok {
			continue
		}

		_, partOfPrimaryKey := field.Tag.Lookup(internal.TagPrimaryKey)

		spec := internal.ColumnSpec(field.Tag)
		metaC := &metaColumn{
			Field:            field,
			OmitEmpty:        spec.OmitEmpty(),
			PartOfPrimaryKey: partOfPrimaryKey,
			ColumnName:       internal.FirstNotEmpty(spec.Name(), strcase.SnakeCase(field.Name)),
		}
		if _, refeField := field.Tag.Lookup(internal.TagForeignKey); !refeField {
			metaT.Columns = append(metaT.Columns, metaC)
		}
		if partOfPrimaryKey {
			metaT.PrimaryKey = append(metaT.PrimaryKey, metaC)
		}
		metaT.RefecenceKeys = m.computeReferenceKeysOf(typ)
	}
	return metaT
}

func (m *MetaSchema) structFieldsOf(typ reflect.Type) []reflect.StructField {
	if typ.Kind() != reflect.Struct {
		panic(fmt.Sprintf("goen: only accepts struct type, but got %q", typ))
	}

	fields := []reflect.StructField{}
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Anonymous {
			embeddedTyp := field.Type
			for embeddedTyp.Kind() == reflect.Ptr {
				embeddedTyp = embeddedTyp.Elem()
			}
			// only support embedded struct
			if embeddedTyp.Kind() == reflect.Struct {
				fields = append(fields, m.structFieldsOf(embeddedTyp)...)
			}
		} else {
			fields = append(fields, field)
		}
	}
	return fields
}

func (m *MetaSchema) computeReferenceKeysOf(typ reflect.Type) [][]*metaColumn {
	var keys [][]*metaColumn
	for _, refeTyp := range m.typlist {
		var key []*metaColumn
		for _, refeField := range m.structFieldsOf(refeTyp) {
			ftyp := refeField.Type
			for ftyp.Kind() == reflect.Ptr || ftyp.Kind() == reflect.Slice {
				ftyp = ftyp.Elem()
			}
			if _, ok := refeField.Tag.Lookup(internal.TagForeignKey); !ok {
				// not a reference
				continue
			} else if ftyp != typ {
				// not a reference
				continue
			}
			spec := internal.ForeignKeySpec(refeField.Tag)
			// child key is typ's key
			for _, childKey := range spec.ChildKey() {
				for _, childField := range m.structFieldsOf(typ) {
					if _, ok := childField.Tag.Lookup(internal.TagIgnore); ok {
						continue
					}
					childSpec := internal.ColumnSpec(childField.Tag)
					columnName := internal.FirstNotEmpty(childSpec.Name(), strcase.SnakeCase(childField.Name))
					if columnName != childKey {
						continue
					}
					_, partOfPrimaryKey := childField.Tag.Lookup(internal.TagPrimaryKey)
					key = append(key, &metaColumn{
						Field:            childField,
						ColumnName:       columnName,
						PartOfPrimaryKey: partOfPrimaryKey,
						OmitEmpty:        childSpec.OmitEmpty(),
					})
				}
			}
		}
		if len(key) > 0 {
			keys = append(keys, key)
		}
	}
	return keys
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
