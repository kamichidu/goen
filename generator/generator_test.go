package generator

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerator(t *testing.T) {
	t.Run("handling pointer field", func(t *testing.T) {
		g := &Generator{
			SrcDir: "./testdata/",
			SrcFileFilter: func(info os.FileInfo) bool {
				return info.Name() == "pointer.go"
			},
		}
		if !assert.NoError(t, g.ParseDir()) {
			return
		}
		assert.Equal(t, &Package{
			PackageName: "testing",
			Imports: append(requiredImports, []*Import{
				&Import{
					Name: "time",
					Path: "time",
				},
			}...),
			Tables: []*Table{
				&Table{
					TableName: "record",
					Entity:    "Record",
					Columns: []*Column{
						&Column{
							ColumnName: "value",
							FieldName:  "Value",
							FieldType:  "*time.Time",
						},
					},
				},
			},
		}, g.pkgData)
	})
	t.Run("handling package alias", func(t *testing.T) {
		g := &Generator{
			SrcDir: "./testdata/",
			SrcFileFilter: func(info os.FileInfo) bool {
				return info.Name() == "pkg_alias.go"
			},
		}
		if !assert.NoError(t, g.ParseDir()) {
			return
		}
		assert.Equal(t, &Package{
			PackageName: "testing",
			Imports: append(requiredImports, []*Import{
				&Import{
					Name: "time",
					Path: "time",
				},
				&Import{
					Name: "github_com_satori_go_uuid",
					Path: "github.com/satori/go.uuid",
				},
			}...),
			Tables: []*Table{
				&Table{
					TableName: "record",
					Entity:    "Record",
					Columns: []*Column{
						&Column{
							ColumnName: "value",
							FieldName:  "Value",
							FieldType:  "*time.Time",
						},
						&Column{
							ColumnName: "value2",
							FieldName:  "Value2",
							FieldType:  "github_com_satori_go_uuid.UUID",
						},
						&Column{
							ColumnName: "value3",
							FieldName:  "Value3",
							FieldType:  "github_com_satori_go_uuid.UUID",
						},
					},
				},
			},
		}, g.pkgData)
	})
}
