package generator

import (
	"encoding/json"
	"fmt"
	"github.com/kamichidu/goen/internal"
	"github.com/kamichidu/goen/internal/asts"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"log"
	"os"
	"reflect"
	"sort"
	"strings"
)

type Generator struct {
	OutPkgName string

	SrcDir string

	SrcFileFilter func(os.FileInfo) bool

	pkgData *Package
}

func importAstPkg(imports map[string]*ast.Object, path string) (pkg *ast.Object, err error) {
	if pkg, ok := imports[path]; ok {
		return pkg, nil
	}
	// find pkg dir and its name
	bpkg, err := build.Import(path, "", build.AllowBinary)
	if err != nil {
		log.Printf("importing %s error: %s", path, err)
		return nil, err
	}
	fset := token.NewFileSet()
	apkgs, err := parser.ParseDir(fset, bpkg.Dir, nil, parser.AllErrors)
	if err != nil {
		log.Printf("parsing directory %s error: %s", bpkg.Dir, err)
		return nil, err
	}
	apkg, ok := apkgs[bpkg.Name]
	if !ok {
		panic("goen: no `" + bpkg.Name + "` package found on " + bpkg.Dir)
	}
	// see https://golang.org/pkg/go/ast/#Object
	scope := ast.NewScope(nil)
	for _, file := range apkg.Files {
		for _, obj := range file.Scope.Objects {
			scope.Insert(obj)
		}
	}
	pkg = ast.NewObj(ast.Pkg, apkg.Name)
	pkg.Data = scope
	imports[path] = pkg
	return pkg, nil
}

var aUniverse *ast.Scope

func init() {
	verbose := debug != ""
	tscope := types.Universe
	ascope := ast.NewScope(nil)
	for _, name := range tscope.Names() {
		var aobj *ast.Object
		switch tobj := tscope.Lookup(name).(type) {
		case *types.TypeName:
			if verbose {
				log.Printf("types.TypeName = (%s, %s)", tobj.Id(), tobj.Name())
			}
			aobj = ast.NewObj(ast.Typ, tobj.Name())
		case *types.Builtin:
			if verbose {
				log.Printf("types.Builtin = (%s, %s)", tobj.Id(), tobj.Name())
			}
			aobj = ast.NewObj(ast.Fun, tobj.Name())
		case *types.Const:
			if verbose {
				log.Printf("types.Const = (%s, %s)", tobj.Id(), tobj.Name())
			}
			aobj = ast.NewObj(ast.Con, tobj.Name())
		case *types.Nil:
			if verbose {
				log.Printf("types.Nil = (%s, %s)", tobj.Id(), tobj.Name())
			}
			aobj = ast.NewObj(ast.Var, tobj.Name())
		default:
			panic(fmt.Sprintf("goen: unknown types.Object %T", tobj))
		}
		ascope.Insert(aobj)
	}
	aUniverse = ascope
}

func (g *Generator) ParseDir() error {
	g.pkgData = new(Package)

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, g.SrcDir, g.SrcFileFilter, parser.AllErrors)
	if err != nil {
		return err
	} else if len(pkgs) > 1 {
		pkgName, _ := asts.AssumeImport(g.SrcDir)
		if pkg, ok := pkgs[pkgName]; ok {
			pkgs = map[string]*ast.Package{
				pkgName: pkg,
			}
		} else {
			var choices []string
			for pkgName := range pkgs {
				choices = append(choices, pkgName)
			}
			return fmt.Errorf("goen: anable to detect package: choices %v", strings.Join(choices, ", "))
		}
	}

	var pkg *ast.Package
	for _, pkg = range pkgs {
		break
	}
	log.Printf("analyzing package %q", pkg.Name)

	// pkg, err = ast.NewPackage(fset, pkg.Files, importAstPkg, aUniverse)
	// if err != nil {
	// 	log.Printf("new package `%s` error: %s", pkg.Name, err)
	// 	return err
	// }

	if err := g.walkPkg(pkg); err != nil {
		return err
	}

	// keep order for idempotency
	entityNames := []string{}
	tables := map[string]*Table{}
	for _, tbl := range g.pkgData.Tables {
		entityNames = append(entityNames, tbl.Entity)
		tables[tbl.Entity] = tbl
	}
	sort.Strings(entityNames)
	g.pkgData.Tables = nil
	for _, entityName := range entityNames {
		g.pkgData.Tables = append(g.pkgData.Tables, tables[entityName])
	}
	return nil
}

func (g *Generator) Generate(w io.Writer) error {
	if debug != "" {
		enc := json.NewEncoder(os.Stderr)
		enc.SetIndent("", "  ")
		if err := enc.Encode(g.pkgData); err != nil {
			log.Printf("write verbose output failed: %s", err)
		}
	}
	return templates.ExecuteTemplate(w, "root.tgo", g.pkgData)
}

func (g *Generator) addImport(path string) {
	g.addImportAs("", path)
}

func (g *Generator) addImportAs(name string, path string) {
	for _, other := range g.pkgData.Imports {
		if other.Path == path {
			return
		}
	}
	g.pkgData.Imports = append(g.pkgData.Imports, &Import{
		Name: name,
		Path: path,
	})
}

func (g *Generator) walkPkg(pkg *ast.Package) error {
	if g.OutPkgName != "" {
		g.pkgData.PackageName = g.OutPkgName
	} else {
		g.pkgData.PackageName = pkg.Name
	}

	// required imports
	g.addImport("database/sql")
	g.addImport("gopkg.in/Masterminds/squirrel.v1")
	g.addImport("github.com/kamichidu/goen")
	if g.OutPkgName != "" {
		// import src package containing entities
		_, pkgPath := asts.AssumeImport(g.SrcDir)
		g.addImportAs(".", pkgPath)
	}

	for filename, file := range pkg.Files {
		log.Printf("analyzing file %q", filename)

		if err := g.walkFile(pkg, file); err != nil {
			return err
		}
	}
	return nil
}

func (g *Generator) walkFile(pkg *ast.Package, file *ast.File) error {
	for _, obj := range file.Scope.Objects {
		if !asts.IsExported(obj) {
			continue
		} else if !asts.IsStructObject(obj) {
			continue
		}

		strct := internal.NewStructFromAST(pkg, file, obj)
		_, ok := internal.FieldByFunc(strct.Fields(), func(field internal.StructField) bool {
			_, ok := field.Tag().Lookup(internal.TagGoen)
			return ok
		})
		if !ok {
			continue
		}

		log.Printf("analyzing struct type %q", strct.Name())
		if err := g.walkStruct(strct); err != nil {
			return err
		}
	}
	return nil
}

func (g *Generator) walkStruct(strct internal.Struct) error {
	tbl := new(Table)
	tbl.TableName = internal.TableName(strct)
	tbl.ReadOnly = internal.IsViewStruct(strct)
	tbl.Entity = strct.Name()

	colFields := internal.FieldsByFunc(strct.Fields(), internal.IsColumnField)
	for _, field := range colFields {
		log.Printf("analyzing struct field %s.%s as column", strct.Name(), field.Name())
		col := new(Column)
		col.ColumnName = internal.ColumnName(field)
		col.OmitEmpty = internal.OmitEmpty(field)
		col.IsPK = internal.IsPrimaryKeyField(field)
		col.FieldName = field.Name()
		col.FieldType = field.Type().String()
		tbl.Columns = append(tbl.Columns, col)
		// when typ is pointer, typ.PkgPath() returns empty string.
		typ := field.Type()
		for typ.Kind() == reflect.Ptr {
			typ = typ.Elem()
		}
		if typ.PkgPath() != "" {
			g.addImport(typ.PkgPath())
		}
	}
	foreFields := internal.FieldsByFunc(strct.Fields(), func(field internal.StructField) bool {
		return !internal.IsIgnoredField(field) && internal.IsForeignKeyField(field)
	})
	for _, field := range foreFields {
		log.Printf("analyzing struct field %s.%s as relation", strct.Name(), field.Name())
		var refeStrct internal.Struct
		switch {
		case internal.IsOneToManyField(field):
			log.Printf("found one-to-many reference field %s.%s", strct.Name(), field.Name())
			typ := field.Type()
			for typ.Kind() == reflect.Ptr || typ.Kind() == reflect.Slice {
				typ = typ.Elem()
			}
			refeStrct = typ.NewStruct()
		case internal.IsManyToOneField(field):
			log.Printf("found many-to-one reference field %s.%s", strct.Name(), field.Name())
			typ := field.Type()
			for typ.Kind() == reflect.Ptr {
				typ = typ.Elem()
			}
			refeStrct = typ.NewStruct()
		default:
			// TODO
			log.Printf("found one-to-one reference field %s.%s", strct.Name(), field.Name())
			typ := field.Type()
			for typ.Kind() == reflect.Ptr {
				typ = typ.Elem()
			}
			refeStrct = typ.NewStruct()
		}
		if refeStrct == nil {
			panic("goen: refeStrct is nil")
		}
		refeFields := internal.FieldsByFunc(refeStrct.Fields(), internal.IsColumnField)
		rel := &Relation{
			FieldName: field.Name(),
			FieldType: refeStrct.Name(),
			TableName: internal.TableName(refeStrct),
		}
		for _, refeField := range refeFields {
			rel.ColumnNames = append(rel.ColumnNames, internal.ColumnName(refeField))
		}
		for _, foreColName := range internal.ForeignKey(field) {
			foreField, ok := internal.FieldByFunc(colFields, internal.EqColumnName(foreColName))
			if !ok {
				panic(fmt.Sprintf("goen: invalid column name found on %s.%s", strct.Name(), field.Name()))
			}
			rel.ForeignKeys = append(rel.ForeignKeys, &RelationalColumn{
				ColumnName: foreColName,
				FieldName:  foreField.Name(),
				FieldType:  foreField.Type().String(),
			})
		}
		for _, refeColName := range internal.ReferenceKey(field) {
			refeField, ok := internal.FieldByFunc(refeFields, internal.EqColumnName(refeColName))
			if !ok {
				panic(fmt.Sprintf("goen: invalid column name found on %s.%s", strct.Name(), field.Name()))
			}
			rel.References = append(rel.References, &RelationalColumn{
				ColumnName: refeColName,
				FieldName:  refeField.Name(),
				FieldType:  refeField.Type().String(),
			})
		}
		switch {
		case internal.IsOneToManyField(field):
			tbl.OneToManyRelations = append(tbl.OneToManyRelations, rel)
		case internal.IsManyToOneField(field):
			tbl.ManyToOneRelations = append(tbl.ManyToOneRelations, rel)
		default:
			tbl.OneToOneRelations = append(tbl.OneToOneRelations, rel)
		}
	}
	g.pkgData.Tables = append(g.pkgData.Tables, tbl)
	return nil
}
