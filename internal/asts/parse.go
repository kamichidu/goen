package asts

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
)

func ParsePkgPath(pkgPath string) (*ast.Package, error) {
	// resolve directory of pkg path
	bpkg, err := bImport(pkgPath, ".", build.FindOnly)
	if err != nil {
		return nil, err
	}
	// parse resolved directory
	fset := token.NewFileSet()
	apkgs, err := parser.ParseDir(fset, bpkg.Dir, nil, parser.AllErrors)
	if err != nil {
		return nil, err
	}
	apkg, ok := apkgs[AssumePkgName(pkgPath)]
	if !ok {
		return nil, fmt.Errorf("goen: package directory %q is not found", pkgPath)
	}
	return apkg, nil
}
