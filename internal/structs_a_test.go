package internal

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func astTestData(objName string) (*ast.Package, *ast.File, *ast.Object) {
	const dataDir = "./testdata/"
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dataDir, nil, parser.AllErrors)
	if err != nil {
		panic(err)
	}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			if obj := file.Scope.Lookup(objName); obj != nil {
				return pkg, file, obj
			}
		}
	}
	panic(fmt.Sprintf("no such object %q", objName))
}

func TestAStruct(t *testing.T) {
	t.Run("Name", func(t *testing.T) {
		strct := NewStructFromAST(astTestData("Testing"))
		assert.IsType(t, (*aStruct)(nil), strct)
		assert.Equal(t, "Testing", strct.Name())
	})
	t.Run("Fields", func(t *testing.T) {
		t.Run("", func(t *testing.T) {
			strct := NewStructFromAST(astTestData("Testing"))
			fields := strct.Fields()
			var names []string
			for _, field := range fields {
				assert.IsType(t, (*aStructField)(nil), field)
				names = append(names, field.Name())
			}
			assert.Equal(t, []string{"AnkoSuki", "AnkoSokosoko", "AnkoKirai"}, names)
		})
		t.Run("", func(t *testing.T) {
			strct := NewStructFromAST(astTestData("TestingEmbedded"))
			fields := strct.Fields()
			var names []string
			for _, field := range fields {
				assert.IsType(t, (*aStructField)(nil), field)
				names = append(names, field.Name())
			}
			assert.Equal(t, []string{"AnkoSuki", "AnkoSokosoko", "AnkoKirai", "AnkoNanisore"}, names)
		})
	})
	t.Run("Value", func(t *testing.T) {
		strct := NewStructFromAST(astTestData("Testing"))
		assert.Nil(t, strct.Value())
	})
}

func TestAStructField(t *testing.T) {
	t.Run("Name", func(t *testing.T) {})
	t.Run("Type", func(t *testing.T) {})
	t.Run("Tag", func(t *testing.T) {})
	t.Run("Value", func(t *testing.T) {})
}

func TestAType(t *testing.T) {
	t.Run("Name", func(t *testing.T) {
		cases := []struct {
			Expect    string
			FieldName string
		}{
			{"string", "StringDecl"},
			{"int", "IntDecl"},
			{"", "StringPtrDecl"},
			{"", "StringSliceDecl"},
			{"time.Time", "TimeDecl"},
			{"", "TimePtrDecl"},
			{"", "TimeSliceDecl"},
		}
		strct := NewStructFromAST(astTestData("DeclTypes"))
		for _, c := range cases {
			field, ok := FieldByFunc(strct.Fields(), EqFieldName(c.FieldName))
			if !ok {
				assert.Fail(t, "no such field %q in struct %q", c.FieldName, strct.Name())
				continue
			}
			assert.Equal(t, c.Expect, field.Type().Name())
		}
	})
	t.Run("PkgPath", func(t *testing.T) {
		cases := []struct {
			Expect    string
			FieldName string
		}{
			{"", "NoPkgPath"},
			{"github.com/satori/go.uuid", "PkgPath"},
		}
		strct := NewStructFromAST(astTestData("DeclPkgPaths"))
		for _, c := range cases {
			field, ok := FieldByFunc(strct.Fields(), EqFieldName(c.FieldName))
			if !ok {
				assert.Fail(t, "no such field %q in struct %q", c.FieldName, strct.Name())
				continue
			}
			assert.Equal(t, c.Expect, field.Type().PkgPath())
		}
	})
	t.Run("Kind", func(t *testing.T) {
		cases := []struct {
			Expect    reflect.Kind
			FieldName string
		}{
			{reflect.String, "StringDecl"},
			{reflect.Int, "IntDecl"},
			{reflect.Ptr, "StringPtrDecl"},
			{reflect.Slice, "StringSliceDecl"},
		}
		strct := NewStructFromAST(astTestData("DeclTypes"))
		for _, c := range cases {
			field, ok := FieldByFunc(strct.Fields(), EqFieldName(c.FieldName))
			if !ok {
				assert.Fail(t, "no such field %q in struct %q", c.FieldName, strct.Name())
				continue
			}
			assert.Equal(t, c.Expect, field.Type().Kind())
		}
	})
	t.Run("Elem", func(t *testing.T) {
		strct := NewStructFromAST(astTestData("DeclTypes"))
		fieldName := "TimeSlicePtrDecl"
		field, ok := FieldByFunc(strct.Fields(), EqFieldName(fieldName))
		if !ok {
			assert.Fail(t, "no such field %q in struct %q", fieldName, strct.Name())
			return
		}
		typ := field.Type()
		assert.Equal(t, "*[]time.Time", typ.String())
		typ = typ.Elem()
		assert.Equal(t, "[]time.Time", typ.String())
		typ = typ.Elem()
		assert.Equal(t, "time.Time", typ.String())
		assert.Panics(t, func() {
			typ.Elem()
		})
	})
	t.Run("String", func(t *testing.T) {
		cases := []struct {
			Expect    string
			FieldName string
		}{
			{"string", "StringDecl"},
			{"int", "IntDecl"},
			{"*string", "StringPtrDecl"},
			{"[]string", "StringSliceDecl"},
			{"[]*string", "StringPtrSliceDecl"},
			{"*[]string", "StringSlicePtrDecl"},
			{"time.Time", "TimeDecl"},
			{"*time.Time", "TimePtrDecl"},
			{"[]time.Time", "TimeSliceDecl"},
			{"[]*time.Time", "TimePtrSliceDecl"},
			{"*[]time.Time", "TimeSlicePtrDecl"},
		}
		strct := NewStructFromAST(astTestData("DeclTypes"))
		for _, c := range cases {
			field, ok := FieldByFunc(strct.Fields(), EqFieldName(c.FieldName))
			if !ok {
				assert.Fail(t, "no such field %q in struct %q", c.FieldName, strct.Name())
				continue
			}
			assert.Equal(t, c.Expect, field.Type().String())
		}
	})
	t.Run("Value", func(t *testing.T) {
		strct := NewStructFromAST(astTestData("Testing"))
		f, ok := FieldByFunc(strct.Fields(), EqFieldName("AnkoSuki"))
		if !ok {
			panic("no such field")
		}
		typ := f.Type()
		assert.Nil(t, typ.Value())
	})
}
