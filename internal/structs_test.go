package internal

import (
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

type testingStruct struct {
	name string

	fields []StructField
}

func (strct *testingStruct) Name() string {
	return strct.name
}

func (strct *testingStruct) Fields() []StructField {
	return strct.fields
}

func (strct *testingStruct) Value() interface{} {
	return nil
}

type testingStructField struct {
	name string

	typ reflect.Type

	tag string
}

func (tsf *testingStructField) Name() string {
	return tsf.name
}

func (tsf *testingStructField) Type() Type {
	return &rType{tsf.typ}
}

func (tsf *testingStructField) Tag() reflect.StructTag {
	return reflect.StructTag(tsf.tag)
}

func (tsf *testingStructField) Value() interface{} {
	return nil
}

func TestTableName(t *testing.T) {
	cases := []struct {
		TableName string
		Struct    Struct
	}{
		{"blog", &testingStruct{name: "blog"}},
		{"blogs", &testingStruct{name: "blog", fields: []StructField{
			&testingStructField{tag: `table:"blogs"`},
		}}},
	}
	for _, c := range cases {
		assert.Equal(t, c.TableName, TableName(c.Struct))
	}
}

func TestIsColumnField(t *testing.T) {
	cases := []struct {
		Expect bool
		Field  StructField
	}{
		{true, &testingStructField{tag: ``}},
		{true, &testingStructField{tag: `primary_key:""`}},
		{true, &testingStructField{tag: `column:""`}},
		{false, &testingStructField{tag: `foreign_key:""`}},
		{false, &testingStructField{tag: `ignore:""`}},
		{false, &testingStructField{tag: `ignore:"" primary_key:""`}},
		{false, &testingStructField{tag: `ignore:"" column:""`}},
		{false, &testingStructField{tag: `ignore:"" foreign_key:""`}},
	}
	for _, c := range cases {
		assert.Equal(t, c.Expect, IsColumnField(c.Field), "StructField(tag=`%s`)", c.Field.Tag())
	}
}

func TestColumnName(t *testing.T) {
	cases := []struct {
		ColumnName string
		Field      StructField
	}{
		{"anko_suki", &testingStructField{name: "AnkoSuki", tag: ``}},
		{"anko_kirai", &testingStructField{name: "AnkoSuki", tag: `primary_key:"anko_kirai"`}},
		{"anko_kirai", &testingStructField{name: "AnkoSuki", tag: `column:"anko_kirai"`}},
		{"anko_suki", &testingStructField{name: "AnkoSuki", tag: `column:",omitempty"`}},
	}
	for _, c := range cases {
		assert.Equal(t, c.ColumnName, ColumnName(c.Field), "StructField(name=%q tag=`%s`)", c.Field.Name(), c.Field.Tag())
	}
}

func TestOmitEmpty(t *testing.T) {
	cases := []struct {
		OmitEmpty bool
		Field     StructField
	}{
		{false, &testingStructField{tag: ``}},
		{true, &testingStructField{tag: `primary_key:"anko_kirai,omitempty"`}},
		{true, &testingStructField{tag: `column:"anko_kirai,omitempty"`}},
	}
	for _, c := range cases {
		assert.Equal(t, c.OmitEmpty, OmitEmpty(c.Field), "StructField(tag=`%s`)", c.Field.Tag())
	}
}

func TestForeignKey(t *testing.T) {
	cases := []struct {
		ForeignKey []string
		Field      StructField
	}{
		{[]string{"a", "b"}, &testingStructField{tag: `foreign_key:"a,b"`}},
		{[]string{"a", "b"}, &testingStructField{tag: `foreign_key:"a:A,b:B"`}},
	}
	for _, c := range cases {
		assert.Equal(t, c.ForeignKey, ForeignKey(c.Field), "StructField(tag=`%s`)", c.Field.Tag())
	}
}

func TestReferenceKey(t *testing.T) {
	cases := []struct {
		ReferenceKey []string
		Field        StructField
	}{
		{[]string{"a", "b"}, &testingStructField{tag: `foreign_key:"a,b"`}},
		{[]string{"A", "B"}, &testingStructField{tag: `foreign_key:"a:A,b:B"`}},
	}
	for _, c := range cases {
		assert.Equal(t, c.ReferenceKey, ReferenceKey(c.Field), "StructField(tag=`%s`)", c.Field.Tag())
	}
}

func TestEqFieldName(t *testing.T) {
	var ankoKirai StructField = &testingStructField{name: "AnkoKirai"}
	var ankoSuki StructField = &testingStructField{name: "AnkoSuki"}
	fn := EqFieldName("AnkoKirai")
	assert.Equal(t, true, fn(ankoKirai))
	assert.Equal(t, false, fn(ankoSuki))
}

func TestEqColumnName(t *testing.T) {
	var ankoKirai StructField = &testingStructField{name: "AnkoSuki", tag: `column:"anko_kirai"`}
	var ankoSuki StructField = &testingStructField{name: "AnkoSuki", tag: ``}
	fn := EqColumnName("anko_kirai")
	assert.Equal(t, true, fn(ankoKirai))
	assert.Equal(t, false, fn(ankoSuki))
}

func TestFieldByFunc(t *testing.T) {
	fields := []StructField{
		&testingStructField{name: "AnkoSuki", tag: ``},
		&testingStructField{name: "AnkoSuki", tag: `column:"anko_kirai"`},
		&testingStructField{name: "AnkoKirai", tag: `column:"anko_sokosoko"`},
	}
	f, ok := FieldByFunc(fields, EqColumnName("anko_kirai"))
	assert.True(t, ok)
	assert.Exactly(t, fields[1], f)
}

func TestFieldsByFunc(t *testing.T) {
	fields := []StructField{
		&testingStructField{name: "AnkoSuki", tag: ``},
		&testingStructField{name: "AnkoSuki", tag: `column:"anko_kirai"`},
		&testingStructField{name: "AnkoKirai", tag: `column:"anko_sokosoko"`},
		&testingStructField{name: "AnkoKirai", tag: ``},
	}
	filtered := FieldsByFunc(fields, EqColumnName("anko_kirai"))
	assert.Exactly(t, []StructField{fields[1], fields[3]}, filtered)
}

func TestIsIgnoredField(t *testing.T) {
	cases := []struct {
		Expect bool
		Field  StructField
	}{
		{false, &testingStructField{tag: ``}},
		{true, &testingStructField{tag: `ignore:""`}},
	}
	for _, c := range cases {
		assert.Equal(t, c.Expect, IsIgnoredField(c.Field), "StructField(tag=`%s`)", c.Field.Tag())
	}
}

func TestIsPrimaryKeyField(t *testing.T) {
	cases := []struct {
		Expect bool
		Field  StructField
	}{
		{false, &testingStructField{tag: ``}},
		{false, &testingStructField{tag: `column:""`}},
		{true, &testingStructField{tag: `primary_key:""`}},
	}
	for _, c := range cases {
		assert.Equal(t, c.Expect, IsPrimaryKeyField(c.Field), "StructField(tag=`%s`)", c.Field.Tag())
	}
}

func TestIsForeignKeyField(t *testing.T) {
	cases := []struct {
		Expect bool
		Field  StructField
	}{
		{false, &testingStructField{tag: ``}},
		{true, &testingStructField{tag: `foreign_key:""`}},
	}
	for _, c := range cases {
		assert.Equal(t, c.Expect, IsForeignKeyField(c.Field), "StructField(tag=`%s`)", c.Field.Tag())
	}
}
