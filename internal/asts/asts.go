package asts

import (
	"fmt"
	"go/ast"
	"go/build"
	"path/filepath"
	"regexp"
	"strings"
)

var rNonWord = regexp.MustCompile(`\W`)

type VisitorFunc func(ast.Node) ast.Visitor

func (fn VisitorFunc) Visit(node ast.Node) ast.Visitor {
	return fn(node)
}

func IsStructObject(obj *ast.Object) bool {
	typSpec, ok := obj.Decl.(*ast.TypeSpec)
	if !ok {
		return false
	}
	_, ok = typSpec.Type.(*ast.StructType)
	return ok
}

func IsExported(obj *ast.Object) bool {
	return ast.IsExported(obj.Name)
}

func EqObjectName(name string) func(*ast.Object) bool {
	return func(obj *ast.Object) bool {
		return obj.Name == name
	}
}

func ObjectByFunc(pkg *ast.Package, fn func(*ast.Object) bool) (*ast.File, *ast.Object, bool) {
	for _, file := range pkg.Files {
		for _, obj := range file.Scope.Objects {
			if fn(obj) {
				return file, obj, true
			}
		}
	}
	return nil, nil, false
}

func GetPkgName(decl ast.Expr) (pkgName string) {
	var visitor ast.Visitor
	visitor = VisitorFunc(func(node ast.Node) ast.Visitor {
		switch expr := node.(type) {
		case *ast.SelectorExpr:
			// e.g. x time.Time
			pkgName = expr.X.(*ast.Ident).Name
			return nil
		}
		return visitor
	})
	ast.Walk(visitor, decl)
	return pkgName
}

func GetTypeName(decl ast.Expr) (typName string) {
	var visitor ast.Visitor
	visitor = VisitorFunc(func(node ast.Node) ast.Visitor {
		switch expr := node.(type) {
		case *ast.Ident:
			// e.g. x string
			// e.g. x SamePackageType
			typName = expr.Name
			return nil
		case *ast.SelectorExpr:
			// e.g. x time.Time
			typName = expr.Sel.Name
			return nil
		}
		return visitor
	})
	ast.Walk(visitor, decl)
	return typName
}

func FindPkgPath(file *ast.File, pkgName string) string {
	var pkgPath string
	for _, impSpec := range file.Imports {
		if impSpec.Name != nil {
			// e.g. uuid "github.com/satori/go.uuid"
			if impSpec.Name.Name == pkgName {
				pkgPath = impSpec.Path.Value
			}
		} else {
			// e.g. "github.com/satori/go.uuid"
			asmPkgName := AssumePkgName(strings.Trim(impSpec.Path.Value, `"`))
			if asmPkgName == pkgName {
				pkgPath = impSpec.Path.Value
			}
		}
	}
	if pkgPath == "" {
		panic(fmt.Sprintf("goen: unable to find package path in %q: %q", file.Name.Name, pkgName))
	}
	return strings.Trim(pkgPath, `"`)
}

func AssumePkgName(pkgPath string) string {
	n := rNonWord.Split(pkgPath, -1)
	return n[len(n)-1]
}

func AssumeImport(dir string) (pkgName string, pkgPath string) {
	absdir, err := filepath.Abs(dir)
	if err != nil {
		panic(fmt.Sprintf("goen: unable to make absolute directory path %q: %s", dir, err))
	}

	bpkg, err := bImportDir(absdir, build.ImportComment)
	if err == nil {
		return bpkg.Name, bpkg.ImportPath
	} else if _, ok := err.(*build.NoGoError); !ok {
		panic(fmt.Sprintf("goen: unable to import directory %q: %s", absdir, err))
	}

	// XXX: when no go file error, modify filepath to assume pkgName and pkgPath
	pkgName = filepath.Base(dir)
	for _, goDir := range build.Default.SrcDirs() {
		// filepath.HasPrefix is now deprecated, but ignore linter for this line.
		// in the future, should implements correct function.
		if !filepath.HasPrefix(absdir, goDir) { // nolint: staticcheck
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
