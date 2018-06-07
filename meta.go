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

func (m *MetaSchema) RowKeysOf(entity interface{}) (primary RowKey, refes []RowKey) {
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
	primary = rowKey

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
	metaT := &metaTable{
		Typ: typ,
	}
	for i := 0; i < typ.NumField(); i++ {
		sf := typ.Field(i)
		if _, ok := internal.FirstLookup(sf.Tag, internal.TagTable, internal.TagView); ok {
			spec := internal.TableSpec(sf.Tag)
			metaT.TableName = spec.Name()
		} else if _, ok := sf.Tag.Lookup(internal.TagIgnore); ok {
			continue
		}

		spec := internal.ColumnSpec(sf.Tag)
		metaC := &metaColumn{
			Field:      sf,
			OmitEmpty:  spec.OmitEmpty(),
			ColumnName: internal.FirstNotEmpty(spec.Name(), strcase.SnakeCase(sf.Name)),
		}
		metaT.Columns = append(metaT.Columns, metaC)

		if _, ok := sf.Tag.Lookup(internal.TagPrimaryKey); ok {
			metaT.PrimaryKey = append(metaT.PrimaryKey, metaC)
		}
		metaT.RefecenceKeys = m.computeReferenceKeysOf(typ)
	}
	return metaT
}

func (m *MetaSchema) computeReferenceKeysOf(typ reflect.Type) [][]*metaColumn {
	var keys [][]*metaColumn
	for _, refeTyp := range m.typlist {
		var key []*metaColumn
		for i := 0; i < refeTyp.NumField(); i++ {
			sf := refeTyp.Field(i)
			ftyp := sf.Type
			for ftyp.Kind() == reflect.Ptr || ftyp.Kind() == reflect.Slice {
				ftyp = ftyp.Elem()
			}
			if _, ok := sf.Tag.Lookup(internal.TagForeignKey); !ok {
				// not a reference
				continue
			} else if ftyp != typ {
				// not a reference
				continue
			}
			spec := internal.ForeignKeySpec(sf.Tag)
			// child key is typ's key
			for _, childKey := range spec.ChildKey() {
				for j := 0; j < typ.NumField(); j++ {
					childField := typ.Field(j)
					if _, ok := childField.Tag.Lookup(internal.TagIgnore); ok {
						continue
					}
					childSpec := internal.ColumnSpec(childField.Tag)
					columnName := internal.FirstNotEmpty(childSpec.Name(), strcase.SnakeCase(childField.Name))
					if columnName != childKey {
						continue
					}
					key = append(key, &metaColumn{
						Field:      childField,
						ColumnName: columnName,
						OmitEmpty:  childSpec.OmitEmpty(),
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
