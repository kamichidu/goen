package internal

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRStruct(t *testing.T) {
	type Testing struct {
		AnkoSuki string

		AnkoSokosoko string `ignore:""`

		AnkoKirai string
	}

	type TestingEmbedded struct {
		*Testing

		AnkoNanisore string
	}

	t.Run("NewStructFromReflect", func(t *testing.T) {
		strct := NewStructFromReflect(reflect.TypeOf(Testing{}))
		assert.IsType(t, (*rStruct)(nil), strct)

		assert.Panics(t, func() {
			NewStructFromReflect(reflect.TypeOf(new(Testing)))
		}, "panics when not struct type was given")
	})
	t.Run("Name", func(t *testing.T) {
		strct := NewStructFromReflect(reflect.TypeOf(Testing{}))
		assert.Equal(t, "Testing", strct.Name())
	})
	t.Run("Fields", func(t *testing.T) {
		t.Run("", func(t *testing.T) {
			strct := NewStructFromReflect(reflect.TypeOf(Testing{}))
			fields := strct.Fields()
			var names []string
			for _, field := range fields {
				assert.IsType(t, (*rStructField)(nil), field)
				names = append(names, field.Name())
			}
			assert.Equal(t, []string{"AnkoSuki", "AnkoSokosoko", "AnkoKirai"}, names)
		})
		t.Run("", func(t *testing.T) {
			strct := NewStructFromReflect(reflect.TypeOf(TestingEmbedded{}))
			fields := strct.Fields()
			var names []string
			for _, field := range fields {
				assert.IsType(t, (*rStructField)(nil), field)
				names = append(names, field.Name())
			}
			assert.Equal(t, []string{"AnkoSuki", "AnkoSokosoko", "AnkoKirai", "AnkoNanisore"}, names)
		})
	})
	t.Run("Value", func(t *testing.T) {
		typ := reflect.TypeOf(Testing{})
		strct := NewStructFromReflect(typ)
		assert.Exactly(t, typ, strct.Value())
	})
}

func TestRStructField(t *testing.T) {
	type Testing struct {
		AnkoSuki string `column:"anko_kirai"`
	}

	typ := reflect.TypeOf(Testing{})
	rf, _ := typ.FieldByName("AnkoSuki")
	field := &rStructField{rf}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "AnkoSuki", field.Name())
	})
	t.Run("Type", func(t *testing.T) {
		assert.Equal(t, &rType{rf.Type}, field.Type())
	})
	t.Run("Tag", func(t *testing.T) {
		assert.Equal(t, reflect.StructTag(`column:"anko_kirai"`), field.Tag())
	})
	t.Run("Value", func(t *testing.T) {
		assert.Exactly(t, rf, field.Value())
	})
}

func TestRType(t *testing.T) {
	raw := reflect.TypeOf(reflect.Value{})
	praw := reflect.PtrTo(raw)
	typ := &rType{raw}
	ptyp := &rType{praw}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "Value", typ.Name())
		assert.Equal(t, "", ptyp.Name())
	})
	t.Run("PkgPath", func(t *testing.T) {
		assert.Equal(t, "reflect", typ.PkgPath())
		assert.Equal(t, "", ptyp.PkgPath())
	})
	t.Run("Kind", func(t *testing.T) {
		assert.Equal(t, reflect.Struct, typ.Kind())
		assert.Equal(t, reflect.Ptr, ptyp.Kind())
	})
	t.Run("Elem", func(t *testing.T) {
		assert.Equal(t, typ, ptyp.Elem())

		assert.Panics(t, func() {
			typ.Elem()
		})
	})
	t.Run("Value", func(t *testing.T) {
		assert.Exactly(t, raw, typ.Value())
		assert.Exactly(t, praw, ptyp.Value())
	})
}
