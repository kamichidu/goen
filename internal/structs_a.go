package internal

import (
	"fmt"
	"go/ast"
	"reflect"
	"strings"

	"github.com/kamichidu/goen/internal/asts"
)

type aStruct struct {
	pkg *ast.Package

	file *ast.File

	typSpec *ast.TypeSpec

	strct *ast.StructType
}

func NewStructFromAST(pkg *ast.Package, file *ast.File, obj *ast.Object) Struct {
	typSpec := obj.Decl.(*ast.TypeSpec)
	strct := typSpec.Type.(*ast.StructType)
	return &aStruct{pkg, file, typSpec, strct}
}

func (astrct *aStruct) Name() string {
	return astrct.typSpec.Name.Name
}

func (astrct *aStruct) Fields() (fields []StructField) {
	for _, field := range astrct.strct.Fields.List {
		if field.Names == nil {
			// anonymous field
			var embPkg *ast.Package
			embTypName := asts.GetTypeName(field.Type)
			if pkgName := asts.GetPkgName(field.Type); pkgName != "" {
				// other package
				pkgPath := asts.FindPkgPath(astrct.file, pkgName)
				pkg, err := asts.ParsePkgPath(pkgPath)
				if err != nil {
					panic(fmt.Sprintf("goen: unable to parse package %q: %s", pkgPath, err))
				}
				embPkg = pkg
			} else {
				// maybe, its decl on the same package
				embPkg = astrct.pkg
			}
			if embPkg == nil {
				panic("goen: embPkg is nil")
			}
			embFile, embObj, ok := asts.ObjectByFunc(embPkg, asts.EqObjectName(embTypName))
			if !ok {
				panic(fmt.Sprintf("goen: unable to find object %q in package %q", embTypName, embPkg.Name))
			}
			embStrct := NewStructFromAST(embPkg, embFile, embObj).(*aStruct)
			fields = append(fields, embStrct.Fields()...)
		} else {
			for _, name := range field.Names {
				if !ast.IsExported(name.Name) {
					continue
				}
				fields = append(fields, &aStructField{astrct, field, name})
			}
		}
	}
	return fields
}

func (astrct *aStruct) Value() interface{} {
	return nil
}

type aStructField struct {
	astrct *aStruct

	field *ast.Field

	name *ast.Ident
}

func (afield *aStructField) Name() string {
	return afield.name.Name
}

func (afield *aStructField) Type() Type {
	return newAType(afield.astrct.pkg, afield.astrct.file, afield.field.Type)
}

func (afield *aStructField) Tag() reflect.StructTag {
	tag := afield.field.Tag
	if tag == nil {
		return reflect.StructTag("")
	}
	// trim `table:""` => table:""
	return reflect.StructTag(strings.Trim(tag.Value, "`"))
}

func (afield *aStructField) Value() interface{} {
	return nil
}

type aType struct {
	// package object that has decl
	pkg *ast.Package

	// file object that has decl
	file *ast.File

	// type declaration expr
	decl ast.Expr
}

func newAType(pkg *ast.Package, file *ast.File, decl ast.Expr) *aType {
	return &aType{pkg, file, decl}
}

func (at *aType) Name() string {
	var name string
	visitor := asts.VisitorFunc(func(node ast.Node) ast.Visitor {
		switch expr := node.(type) {
		case *ast.Ident:
			// e.g. x string
			// e.g. x SamePackageType
			name = expr.Name
		case *ast.SelectorExpr:
			// e.g. x time.Time
			pkgName := expr.X.(*ast.Ident).Name
			typName := expr.Sel.Name
			name = pkgName + "." + typName
		}
		return nil
	})
	ast.Walk(visitor, at.decl)
	return name
}

func (at *aType) PkgPath() string {
	var pkgPath string
	visitor := asts.VisitorFunc(func(node ast.Node) ast.Visitor {
		switch expr := node.(type) {
		case *ast.SelectorExpr:
			// e.g. x time.Time
			pkgName := expr.X.(*ast.Ident).Name
			pkgPath = asts.FindPkgPath(at.file, pkgName)
		}
		return nil
	})
	ast.Walk(visitor, at.decl)
	return pkgPath
}

var aPrimitiveKinds = map[string]reflect.Kind{
	"string":  reflect.String,
	"bool":    reflect.Bool,
	"int":     reflect.Int,
	"int8":    reflect.Int8,
	"int16":   reflect.Int16,
	"int32":   reflect.Int32,
	"int64":   reflect.Int64,
	"uint":    reflect.Uint,
	"uint8":   reflect.Uint8,
	"uint16":  reflect.Uint16,
	"uint32":  reflect.Uint32,
	"uint64":  reflect.Uint64,
	"float32": reflect.Float32,
	"float64": reflect.Float64,
}

func (at *aType) Kind() reflect.Kind {
	kind := reflect.Invalid
	visitor := asts.VisitorFunc(func(node ast.Node) ast.Visitor {
		switch expr := node.(type) {
		case *ast.Ident:
			// e.g. x string
			// e.g. x SamePackageType
			if v, ok := aPrimitiveKinds[expr.Name]; ok {
				kind = v
			} else {
				// maybe a type name declared in same package
				defFile, defObj, ok := asts.ObjectByFunc(at.pkg, asts.EqObjectName(expr.Name))
				if !ok {
					panic(fmt.Sprintf("goen: type declaration is not found %q in package %q", expr.Name, at.pkg.Name))
				}
				// must be a type declaration object
				typSpec := defObj.Decl.(*ast.TypeSpec)
				defTyp := newAType(at.pkg, defFile, typSpec.Type)
				kind = defTyp.Kind()
			}
		case *ast.StarExpr:
			// e.g. x *string
			kind = reflect.Ptr
		case *ast.ArrayType:
			kind = reflect.Slice
		case *ast.SelectorExpr:
			// e.g. x time.Time
			pkgName := expr.X.(*ast.Ident).Name
			typName := expr.Sel.Name
			panic(fmt.Sprintf("goen: *ast.SelectorExpr %q.%s", pkgName, typName))
		case *ast.StructType:
			kind = reflect.Struct
		default:
			panic(fmt.Sprintf("goen: unknown expr type %T", expr))
		}
		return nil
	})
	ast.Walk(visitor, at.decl)
	return kind
}

func (at *aType) Elem() Type {
	var elem Type
	visitor := asts.VisitorFunc(func(node ast.Node) ast.Visitor {
		switch expr := node.(type) {
		case *ast.Ident, *ast.SelectorExpr:
			panic("goen: Elem of invalid type")
		case *ast.StarExpr:
			// e.g. x *string
			elem = newAType(at.pkg, at.file, expr.X)
		case *ast.ArrayType:
			elem = newAType(at.pkg, at.file, expr.Elt)
		}
		return nil
	})
	ast.Walk(visitor, at.decl)
	return elem
}

func (at *aType) NewStruct() Struct {
	if at.Kind() != reflect.Struct {
		panic(fmt.Sprintf("goen: NewStruct is only with a struct type, not a %s", at.Kind()))
	}

	var pkg *ast.Package
	if at.PkgPath() != "" {
		var err error
		pkg, err = asts.ParsePkgPath(at.PkgPath())
		if err != nil {
			panic(fmt.Sprintf("goen: failed to parse package %q: %s", at.PkgPath(), err))
		}
	} else {
		pkg = at.pkg
	}
	file, obj, ok := asts.ObjectByFunc(at.pkg, asts.EqObjectName(at.Name()))
	if !ok {
		panic(fmt.Sprintf("goen: couldnot find a struct %q in package %q", at.Name(), pkg.Name))
	}
	return NewStructFromAST(pkg, file, obj)
}

func (at *aType) String() string {
	var s string
	var visitor ast.Visitor
	visitor = asts.VisitorFunc(func(node ast.Node) ast.Visitor {
		switch expr := node.(type) {
		case *ast.Ident:
			// e.g. x string
			// e.g. x SamePackageType
			s += expr.Name
			return nil
		case *ast.StarExpr:
			// e.g. x *string
			s += "*"
			return visitor
		case *ast.ArrayType:
			s += "[]"
			return visitor
		case *ast.SelectorExpr:
			// e.g. x time.Time
			pkgName := expr.X.(*ast.Ident).Name
			typName := expr.Sel.Name
			s += pkgName + "." + typName
			return nil
		default:
			return nil
		}
	})
	ast.Walk(visitor, at.decl)
	return s
}

func (at *aType) Value() interface{} {
	return nil
}
