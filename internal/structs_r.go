package internal

import (
	"fmt"
	"reflect"
)

type rStruct struct {
	reflect.Type
}

func NewStructFromReflect(typ reflect.Type) Struct {
	if typ.Kind() != reflect.Struct {
		panic(fmt.Sprintf("goen: only accepts struct type, but got %q", typ))
	}
	return &rStruct{typ}
}

func (rstrct *rStruct) Fields() (fields []StructField) {
	for i := 0; i < rstrct.Type.NumField(); i++ {
		f := rstrct.Type.Field(i)
		if f.Anonymous {
			embeddedTyp := f.Type
			for embeddedTyp.Kind() == reflect.Ptr {
				embeddedTyp = embeddedTyp.Elem()
			}
			// only support embedded struct
			if embeddedTyp.Kind() == reflect.Struct {
				embedded := NewStructFromReflect(embeddedTyp)
				fields = append(fields, embedded.Fields()...)
			}
		} else {
			fields = append(fields, &rStructField{f})
		}
	}
	return fields
}

func (rstrct *rStruct) Value() interface{} {
	return rstrct.Type
}

type rStructField struct {
	sf reflect.StructField
}

func (rsf *rStructField) Name() string {
	return rsf.sf.Name
}

func (rsf *rStructField) Type() Type {
	return &rType{rsf.sf.Type}
}

func (rsf *rStructField) Tag() reflect.StructTag {
	return rsf.sf.Tag
}

func (rsf *rStructField) Value() interface{} {
	return rsf.sf
}

type rType struct {
	reflect.Type
}

func (rt *rType) Elem() Type {
	return &rType{rt.Type.Elem()}
}

func (rt *rType) Value() interface{} {
	return rt.Type
}
