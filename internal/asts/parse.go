package asts

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
)

func ParsePkgPath(pkgPath string) (*ast.Package, error) {
	// resolve pkgPath as a vendor if possible
	tpkg, err := SrcImporter.Import(pkgPath)
	if err != nil {
		return nil, err
	}
	// resolve directory of pkg path
	bpkg, err := bImport(tpkg.Path(), ".", build.FindOnly)
	if err != nil {
		return nil, err
	}
	// parse resolved directory
	fset := token.NewFileSet()
	apkgs, err := parser.ParseDir(fset, bpkg.Dir, nil, parser.AllErrors)
	if err != nil {
		return nil, err
	}
	apkg, ok := apkgs[tpkg.Name()]
	if !ok {
		return nil, fmt.Errorf("goen: package directory %q is not found", pkgPath)
	}
	return apkg, nil
}
