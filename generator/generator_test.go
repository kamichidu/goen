package generator

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerator(t *testing.T) {
	t.Run("PointerField", func(t *testing.T) {
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
}
