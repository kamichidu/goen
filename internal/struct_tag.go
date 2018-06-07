package internal

import (
	"reflect"
	"strings"
)

const (
	TagGoen       = "goen"
	TagTable      = "table"
	TagView       = "view"
	TagPrimaryKey = "primary_key"
	TagColumn     = "column"
	TagForeignKey = "foreign_key"
	TagIgnore     = "ignore"
)

type TableSpec string

func (s TableSpec) Name() string {
	tv, ok := FirstLookup(reflect.StructTag(s), TagTable, TagView)
	if !ok {
		return ""
	}
	return tv
}

func (s TableSpec) ReadOnly() bool {
	_, ok := reflect.StructTag(s).Lookup(TagView)
	return ok
}

// struct tag example:
//   `primary_key:""`
//   `primary_key:"col"`
//   `primary_key:"col,omitempty"`
//   `primary_key:",omitempty"`
//   `primary_key:","`
//   `column:""`
//   `column:"col"`
//   `column:"col,omitempty"`
//   `column:",omitempty"`
//   `column:","`
type ColumnSpec string

func (s ColumnSpec) Name() string {
	tv, ok := FirstLookup(reflect.StructTag(s), TagPrimaryKey, TagColumn)
	if !ok {
		return ""
	}
	return stringAt(tv, ",", 0)
}

func (s ColumnSpec) OmitEmpty() bool {
	tv, ok := FirstLookup(reflect.StructTag(s), TagPrimaryKey, TagColumn)
	if !ok {
		return false
	}
	return stringAt(tv, ",", 1) == "omitempty"
}

// struct tag example:
//   `foreign_key:"col1,col2"`
//     => foreign key (col1, col2) references another(col1, col2)
//   `foreign_key:"col1,col2:col3"`
//     => foreign key (col1, col2) references another(col1, col3)
type ForeignKeySpec string

// ParentKey is the column or set of columns in the parent table that the foreign key constraint refers to.
func (s ForeignKeySpec) ParentKey() []string {
	pairs := s.colpairs()
	key := make([]string, len(pairs))
	for i, pair := range pairs {
		key[i] = pair[0]
	}
	return key
}

// ChildKey is the column or set of columns in the child table that are constrained by the foreign key constraint and which hold the REFERENCES clause.
func (s ForeignKeySpec) ChildKey() []string {
	pairs := s.colpairs()
	key := make([]string, len(pairs))
	for i, pair := range pairs {
		key[i] = pair[1]
	}
	return key
}

func (s ForeignKeySpec) colpairs() [][]string {
	tv, ok := reflect.StructTag(s).Lookup(TagForeignKey)
	if !ok {
		return nil
	}
	pairs := [][]string{}
	for _, ele := range strings.Split(tv, ",") {
		pkey := stringAt(ele, ":", 0)
		ckey := stringAt(ele, ":", 1)
		if ckey == "" {
			ckey = pkey
		}
		pairs = append(pairs, []string{pkey, ckey})
	}
	return pairs
}

func FirstLookup(tag reflect.StructTag, names ...string) (string, bool) {
	for _, name := range names {
		if s, ok := tag.Lookup(name); ok {
			return s, true
		}
	}
	return "", false
}

func stringAt(s string, sep string, i int) string {
	eles := strings.Split(s, sep)
	if i < len(eles) {
		return eles[i]
	} else {
		return ""
	}
}

func FirstNotEmpty(strs ...string) string {
	for _, s := range strs {
		if s != "" {
			return s
		}
	}
	return ""
}
