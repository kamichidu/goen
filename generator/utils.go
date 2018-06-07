package generator

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/importer"
	"path/filepath"
	"reflect"
	"strings"
)

func isEntityType(obj *ast.Object) bool {
	typSpec, ok := obj.Decl.(*ast.TypeSpec)
	if !ok {
		return false
	}
	expr, ok := typSpec.Type.(*ast.StructType)
	if !ok {
		return false
	}
	for _, f := range expr.Fields.List {
		tag := getStructTag(f)
		if _, ok := tag.Lookup("goen"); ok {
			return true
		}
	}
	return false
}

func findDecl(pkg *ast.Package, typName string) (*ast.File, *ast.Object) {
	for _, file := range pkg.Files {
		if obj := file.Scope.Lookup(typName); obj != nil {
			return file, obj
		}
	}
	return nil, nil
}

func getStructTag(obj *ast.Field) reflect.StructTag {
	if obj.Tag == nil {
		return reflect.StructTag("")
	}
	return reflect.StructTag(strings.Trim(obj.Tag.Value, "`"))
}

type TypeQualifier string

const (
	Ptr   TypeQualifier = "*"
	Array TypeQualifier = "[]"
)

type Type struct {
	PkgPath string

	PkgName string

	Qualifiers []TypeQualifier

	Name string
}

func (typ *Type) IsSlice() bool {
	if len(typ.Qualifiers) == 0 {
		return false
	}
	return typ.Qualifiers[0] == Array
}

func (typ *Type) TypeString() string {
	base := typ.qualifier() + typ.Name
	if typ.PkgName == "" {
		return base
	}
	return typ.PkgName + `.` + base
}

func (typ *Type) String() string {
	base := typ.qualifier() + typ.Name
	if typ.PkgPath == "" {
		return base
	}
	return `"` + typ.PkgPath + `".` + base
}

func (typ *Type) qualifier() string {
	qs := make([]string, len(typ.Qualifiers))
	for i := range typ.Qualifiers {
		qs[i] = string(typ.Qualifiers[i])
	}
	return strings.Join(qs, "")
}

func resolveType(file *ast.File, decl ast.Expr) *Type {
	switch expr := decl.(type) {
	case *ast.Ident:
		// e.g. x string
		// e.g. x SamePackageType
		return &Type{
			Name: expr.Name,
		}
	case *ast.StarExpr:
		// e.g. x *string
		typ := resolveType(file, expr.X)
		typ.Qualifiers = append([]TypeQualifier{Ptr}, typ.Qualifiers...)
		return typ
	case *ast.ArrayType:
		typ := resolveType(file, expr.Elt)
		typ.Qualifiers = append([]TypeQualifier{Array}, typ.Qualifiers...)
		return typ
	case *ast.SelectorExpr:
		// e.g. x time.Time
		pkgName := expr.X.(*ast.Ident).Name
		pkgPath := resolvePackagePath(file, pkgName)
		return &Type{
			PkgPath: pkgPath,
			PkgName: pkgName,
			Name:    expr.Sel.Name,
		}
	default:
		panic(fmt.Sprintf("unsupported decl type: %T", decl))
	}
}

func resolvePackagePath(file *ast.File, sel string) string {
	var pkgPath string
	for _, impSpec := range file.Imports {
		if impSpec.Name != nil {
			// e.g. uuid "github.com/satori/go.uuid"
			if impSpec.Name.Name == sel {
				pkgPath = impSpec.Path.Value
			}
		} else {
			// e.g. "github.com/satori/go.uuid"
			pkgName, err := resolvePackageName(strings.Trim(impSpec.Path.Value, `"`))
			if err != nil {
				panic(fmt.Sprintf("goen: unable to resolve package name of %s: %s", impSpec.Path.Value, err))
			}
			if pkgName == sel {
				pkgPath = impSpec.Path.Value
			}
		}
	}
	if pkgPath == "" {
		panic(fmt.Sprintf("goen: unable to resolve importing package path of %s", sel))
	}
	return strings.Trim(pkgPath, `"`)
}

func resolvePackageName(pkgPath string) (string, error) {
	pkg, err := importer.Default().Import(pkgPath)
	if err != nil {
		return "", err
	}
	return pkg.Name(), nil
}

func AssumeImport(dir string) (pkgName string, pkgPath string) {
	absdir, err := filepath.Abs(dir)
	if err != nil {
		panic(fmt.Sprintf("goen: unable to make absolute directory path %q: %s", dir, err))
	}

	bpkg, err := build.ImportDir(absdir, build.ImportComment)
	if err == nil {
		return bpkg.Name, bpkg.ImportPath
	} else if _, ok := err.(*build.NoGoError); !ok {
		panic(fmt.Sprintf("goen: unable to import directory %q: %s", absdir, err))
	}

	// XXX: when no go file error, modify filepath to assume pkgName and pkgPath
	pkgName = filepath.Base(dir)
	for _, goDir := range build.Default.SrcDirs() {
		if !filepath.HasPrefix(absdir, goDir) {
			continue
		}

		pkgPath, err = filepath.Rel(goDir, absdir)
		if err != nil {
			panic(fmt.Sprintf("goen: unable to get relative path %q to %q: %s", absdir, goDir, err))
		}
		break
	}
	return pkgName, pkgPath
}
