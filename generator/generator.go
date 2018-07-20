package generator

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"log"
	"os"
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
		pkgName, _ := AssumeImport(g.SrcDir)
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
		_, pkgPath := AssumeImport(g.SrcDir)
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
	for objName, obj := range file.Scope.Objects {
		if !ast.IsExported(objName) {
			continue
		} else if !isEntityType(obj) {
			continue
		}

		log.Printf("analyzing struct type %q", objName)
		astrct := newAStructFromObject(pkg, file, obj)
		if err := g.walkStructTypeDecl(astrct); err != nil {
			return err
		}
	}
	return nil
}

func (g *Generator) walkStructTypeDecl(astrct *aStruct) error {
	tbl := new(Table)
	tbl.TableName = astrct.TableName()
	tbl.ReadOnly = astrct.ReadOnly()
	tbl.Entity = astrct.Name()

	for _, afield := range astrct.Fields() {
		if !afield.IsExported() {
			continue
		} else if afield.Ignore() {
			log.Printf("ignore %s.%s", astrct.Name(), afield.Name())
			continue
		}

		log.Printf("analyzing struct field %s.%s", astrct.Name(), afield.Name())
		switch {
		case afield.IsOneToMany(), afield.IsManyToOne(), afield.IsOneToOne():
			if afield.IsOneToMany() {
				log.Printf("found one-to-many reference field %s.%s", astrct.Name(), afield.Name())
			} else if afield.IsManyToOne() {
				log.Printf("found many-to-one reference field %s.%s", astrct.Name(), afield.Name())
			} else {
				// TODO
				log.Printf("found one-to-one reference field %s.%s", astrct.Name(), afield.Name())
			}
			refe := afield.Reference()
			refeColumns := []string{}
			for _, refef := range refe.Fields() {
				if !refef.IsExported() {
					continue
				} else if refef.Ignore() {
					continue
				} else if !refef.IsColumn() {
					continue
				}
				refeColumns = append(refeColumns, refef.ColumnName())
			}
			rel := &Relation{
				FieldName:   afield.Name(),
				FieldType:   refe.Name(),
				TableName:   refe.TableName(),
				ColumnNames: refeColumns,
			}
			for _, colName := range afield.ForeignKeyColumnNames() {
				fkf, ok := astrct.FieldByColumnName(colName)
				if !ok {
					panic(fmt.Sprintf("goen: invalid column name found on %s.%s", astrct.Name(), afield.Name()))
				}
				rel.ForeignKeys = append(rel.ForeignKeys, &RelationalColumn{
					ColumnName: colName,
					FieldName:  fkf.Name(),
					FieldType:  fkf.Type().TypeString(),
				})
			}
			for _, colName := range afield.ReferenceColumnNames() {
				refef, ok := astrct.FieldByColumnName(colName)
				if !ok {
					panic(fmt.Sprintf("goen: invalid column name found on %s.%s", astrct.Name(), afield.Name()))
				}
				rel.References = append(rel.References, &RelationalColumn{
					ColumnName: colName,
					FieldName:  refef.Name(),
					FieldType:  refef.Type().TypeString(),
				})
			}
			if afield.IsOneToMany() {
				tbl.OneToManyRelations = append(tbl.OneToManyRelations, rel)
			} else if afield.IsManyToOne() {
				tbl.ManyToOneRelations = append(tbl.ManyToOneRelations, rel)
			} else {
				tbl.OneToOneRelations = append(tbl.OneToOneRelations, rel)
			}
		case afield.IsColumn():
			col := new(Column)
			col.ColumnName = afield.ColumnName()
			col.OmitEmpty = afield.OmitEmpty()
			col.IsPK = afield.IsPrimaryKey()
			col.FieldName = afield.Name()
			col.FieldType = afield.Type().TypeString()
			tbl.Columns = append(tbl.Columns, col)
			if typ := afield.Type(); typ.PkgPath != "" {
				g.addImport(typ.PkgPath)
			}
		default:
			log.Printf("ignore field %s.%s", astrct.Name(), afield.Name())
		}
	}
	g.pkgData.Tables = append(g.pkgData.Tables, tbl)
	return nil
}
