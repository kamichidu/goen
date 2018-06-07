package generator

import (
	"go/ast"
	"reflect"
	"strings"

	"github.com/kamichidu/goen/internal"
	"github.com/stoewer/go-strcase"
)

type aStruct struct {
	pkg *ast.Package

	file *ast.File

	typSpec *ast.TypeSpec

	strct *ast.StructType
}

func newAStructFromObject(pkg *ast.Package, file *ast.File, obj *ast.Object) *aStruct {
	typSpec := obj.Decl.(*ast.TypeSpec)
	strct := typSpec.Type.(*ast.StructType)
	return &aStruct{pkg, file, typSpec, strct}
}

func (astrct *aStruct) TableName() string {
	for _, afield := range astrct.Fields() {
		spec := internal.TableSpec(afield.StructTag())
		if name := spec.Name(); name != "" {
			return name
		}
	}
	// infer table name
	return strcase.SnakeCase(astrct.Name())
}

func (astrct *aStruct) ReadOnly() bool {
	for _, afield := range astrct.Fields() {
		if _, ok := afield.StructTag().Lookup(internal.TagView); ok {
			return true
		}
	}
	return false
}

func (astrct *aStruct) Name() string {
	return astrct.typSpec.Name.Name
}

func (astrct *aStruct) Fields() []*aStructField {
	fields := []*aStructField{}
	for _, field := range astrct.strct.Fields.List {
		for _, name := range field.Names {
			fields = append(fields, &aStructField{astrct, field, name})
		}
	}
	return fields
}

func (astrct *aStruct) FieldByColumnName(name string) (*aStructField, bool) {
	for _, afield := range astrct.Fields() {
		if afield.ColumnName() == name {
			return afield, true
		}
	}
	return nil, false
}

type aStructField struct {
	astrct *aStruct

	field *ast.Field

	name *ast.Ident
}

func (afield *aStructField) StructTag() reflect.StructTag {
	tag := afield.field.Tag
	if tag == nil {
		return reflect.StructTag("")
	}
	// trim `table:""` => table:""
	return reflect.StructTag(strings.Trim(tag.Value, "`"))
}

func (afield *aStructField) Name() string {
	return afield.name.Name
}

func (afield *aStructField) ColumnName() string {
	spec := internal.ColumnSpec(afield.StructTag())
	return internal.FirstNotEmpty(spec.Name(), strcase.SnakeCase(afield.Name()))
}

func (afield *aStructField) OmitEmpty() bool {
	spec := internal.ColumnSpec(afield.StructTag())
	return spec.OmitEmpty()
}

func (afield *aStructField) IsExported() bool {
	return ast.IsExported(afield.name.Name)
}

func (afield *aStructField) Ignore() bool {
	_, ok := afield.StructTag().Lookup(internal.TagIgnore)
	return ok
}

func (afield *aStructField) IsPrimaryKey() bool {
	_, ok := afield.StructTag().Lookup(internal.TagPrimaryKey)
	return ok
}

// IsOneToMany checks following:
// - field type is a slice of struct that is entity
// - field type is declared on same package
func (afield *aStructField) IsOneToMany() bool {
	typ := afield.Type()
	if typ.PkgPath != "" {
		return false
	}
	if !typ.IsSlice() {
		return false
	}
	_, refeObj := findDecl(afield.astrct.pkg, typ.Name)
	if refeObj == nil {
		return false
	}
	return isEntityType(refeObj)
}

// IsManyToOne checks following:
// - field type is a pointer of struct that is entity
// - field type is declared on same package
func (afield *aStructField) IsManyToOne() bool {
	typ := afield.Type()
	if typ.PkgPath != "" {
		return false
	}
	if typ.IsSlice() {
		return false
	}
	_, refeObj := findDecl(afield.astrct.pkg, typ.Name)
	if refeObj == nil {
		return false
	}
	return isEntityType(refeObj)
}

func (afield *aStructField) IsOneToOne() bool {
	return false
}

// IsColumn checks following:
// - field type is not a slice or array
// - field type is not a entity type
func (afield *aStructField) IsColumn() bool {
	typ := afield.Type()
	if typ.IsSlice() {
		return false
	}
	_, decl := findDecl(afield.astrct.pkg, typ.Name)
	if decl != nil && isEntityType(decl) {
		return false
	}
	return true
}

func (afield *aStructField) Type() *Type {
	return resolveType(afield.astrct.file, afield.field.Type)
}

// Reference returns underlying struct type or nil referenced by this field.
func (afield *aStructField) Reference() *aStruct {
	typ := afield.Type()
	refeFile, refeObj := findDecl(afield.astrct.pkg, typ.Name)
	if !isEntityType(refeObj) {
		return nil
	}
	return newAStructFromObject(afield.astrct.pkg, refeFile, refeObj)
}

func (afield *aStructField) ForeignKeyColumnNames() []string {
	spec := internal.ForeignKeySpec(afield.StructTag())
	return spec.ParentKey()
}

func (afield *aStructField) ReferenceColumnNames() []string {
	spec := internal.ForeignKeySpec(afield.StructTag())
	return spec.ChildKey()
}
