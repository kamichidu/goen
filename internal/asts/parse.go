package asts

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

func ParsePkgPath(pkgPath string) (*ast.Package, error) {
	// resolve directory of pkg path
	bpkg, err := bImport(pkgPath, ".", 0)
	if err != nil {
		return nil, err
	}
	// parse resolved directory
	fset := token.NewFileSet()
	apkgs, err := parser.ParseDir(fset, bpkg.Dir, nil, parser.AllErrors)
	if err != nil {
		return nil, err
	}
	apkg, ok := apkgs[bpkg.Name]
	if !ok {
		return nil, fmt.Errorf("goen: package directory %q is not found", pkgPath)
	}
	return apkg, nil
}
