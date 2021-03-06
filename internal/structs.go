package internal

import (
	"fmt"
	"reflect"

	"github.com/stoewer/go-strcase"
)

// Struct represents go struct type info.
type Struct interface {
	Name() string

	Fields() []StructField

	Value() interface{}
}

// StructField represents go struct field info.
type StructField interface {
	Name() string

	Type() Type

	Tag() reflect.StructTag

	Value() interface{}
}

// Type represents go type info.
type Type interface {
	fmt.Stringer

	Name() string

	// original package path.
	// like github.com/kamichidu/goen, reflect or empty.
	PkgPath() string

	Kind() reflect.Kind

	Elem() Type

	NewStruct() Struct

	Value() interface{}
}

type TypeAlternator interface {
	// stringify with alternative package name.
	// e.g. time.Time => timeAlt.Time
	StringWithPkgName(pkgName string) string
}

func TableName(strct Struct) string {
	name := strcase.SnakeCase(strct.Name())
	for _, field := range strct.Fields() {
		if IsIgnoredField(field) {
			continue
		}
		if _, ok := FirstLookup(field.Tag(), TagTable, TagView); ok {
			spec := TableSpec(field.Tag())
			name = FirstNotEmpty(spec.Name(), name)
			break
		}
	}
	return name
}

func IsViewStruct(strct Struct) bool {
	for _, field := range strct.Fields() {
		if _, ok := field.Tag().Lookup(TagView); ok {
			return true
		}
	}
	return false
}

func IsColumnField(field StructField) bool {
	return !IsIgnoredField(field) && !IsForeignKeyField(field)
}

func ColumnName(field StructField) string {
	if IsIgnoredField(field) {
		panic("goen: unable to get column name from ignored field")
	}
	name := strcase.SnakeCase(field.Name())
	if _, ok := FirstLookup(field.Tag(), TagPrimaryKey, TagColumn); ok {
		spec := ColumnSpec(field.Tag())
		name = FirstNotEmpty(spec.Name(), name)
	}
	return name
}

func OmitEmpty(field StructField) bool {
	if IsIgnoredField(field) {
		panic("goen: unable to get omitempty from ignored field")
	}
	omitEmpty := false
	if _, ok := FirstLookup(field.Tag(), TagPrimaryKey, TagColumn); ok {
		spec := ColumnSpec(field.Tag())
		omitEmpty = spec.OmitEmpty()
	}
	return omitEmpty
}

func ForeignKey(field StructField) []string {
	if !IsForeignKeyField(field) {
		panic("goen: unable to get foreign key from non-foreign key field")
	}
	spec := ForeignKeySpec(field.Tag())
	return spec.ParentKey()
}

func ReferenceKey(field StructField) []string {
	if !IsForeignKeyField(field) {
		panic("goen: unable to get reference key from non-foreign key field")
	}
	spec := ForeignKeySpec(field.Tag())
	return spec.ChildKey()
}

func EqFieldName(name string) func(StructField) bool {
	return func(field StructField) bool {
		return field.Name() == name
	}
}

func EqColumnName(name string) func(StructField) bool {
	return func(field StructField) bool {
		if IsIgnoredField(field) {
			return false
		}
		return ColumnName(field) == name
	}
}

func FieldByFunc(fields []StructField, fn func(StructField) bool) (StructField, bool) {
	for _, field := range fields {
		if fn(field) {
			return field, true
		}
	}
	return nil, false
}

func FieldsByFunc(fields []StructField, fn func(StructField) bool) (filtered []StructField) {
	for _, field := range fields {
		if fn(field) {
			filtered = append(filtered, field)
		}
	}
	return filtered
}

func IsIgnoredField(field StructField) bool {
	_, ok := field.Tag().Lookup(TagIgnore)
	return ok
}

func IsPrimaryKeyField(field StructField) bool {
	_, ok := field.Tag().Lookup(TagPrimaryKey)
	return ok
}

func IsForeignKeyField(field StructField) bool {
	_, ok := field.Tag().Lookup(TagForeignKey)
	return ok
}

func IsOneToManyField(field StructField) bool {
	typ := field.Type()
	if typ.Kind() != reflect.Slice {
		return false
	}
	for typ.Kind() == reflect.Ptr || typ.Kind() == reflect.Slice {
		typ = typ.Elem()
	}
	return mustValidKind(typ.Kind()) == reflect.Struct
}

func IsManyToOneField(field StructField) bool {
	typ := field.Type()
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	return mustValidKind(typ.Kind()) == reflect.Struct
}

func mustValidKind(v reflect.Kind) reflect.Kind {
	if v == reflect.Invalid {
		panic("goen: invalid kind of type")
	}
	return v
}
