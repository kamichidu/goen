package goen

import (
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TextID string

func (s TextID) MarshalText() ([]byte, error) {
	return []byte("MarshalText(" + s + ")"), nil
}

type BlobID string

func (s BlobID) MarshalBinary() ([]byte, error) {
	return []byte("MarshalBinary(" + s + ")"), nil
}

func mustFieldByName(typ reflect.Type, name string) reflect.StructField {
	v, ok := typ.FieldByName(name)
	if !ok {
		panic("no such struct field " + name)
	}
	return v
}

func TestMetadataMap(t *testing.T) {
	type Blog struct {
		IDInt    int    `goen:"" table:"blogs" primary_key:""`
		IDString string `primary_key:""`
		IDTextID TextID `primary_key:""`
		IDBlobID BlobID `primary_key:""`
	}
	type Post struct {
		PostID int `goen:"" table:"posts"`
		BlogID string
		Blog   *Blog `foreign_key:"blog_id:id_string"`
	}
	meta := new(MetaSchema)
	meta.Register(Blog{})
	meta.Register(Post{})
	meta.Compute()

	t.Run("RowKeysOf", func(t *testing.T) {
		blog := &Blog{
			IDInt:    1,
			IDString: "str",
			IDTextID: "tid",
			IDBlobID: "bid",
		}
		pk, refes := meta.RowKeysOf(blog)
		assert.EqualValues(t, &MapRowKey{
			Table: "blogs",
			Key: map[string]interface{}{
				"id_int":     1,
				"id_string":  "str",
				"id_text_id": TextID("tid"),
				"id_blob_id": BlobID("bid"),
			},
		}, pk)
		assert.EqualValues(t, []RowKey{
			&MapRowKey{Table: "blogs", Key: map[string]interface{}{"id_string": "str"}},
		}, refes)
	})
	t.Run("KeyStringOf", func(t *testing.T) {
		key, err := meta.KeyStringFromRowKey(&MapRowKey{
			Table: "blogs",
			Key: map[string]interface{}{
				"id_int":     1,
				"id_string":  "str",
				"id_text_id": TextID("tid"),
				"id_blob_id": BlobID("bid"),
			},
		})
		assert.NoError(t, err)
		// columns are sorted by name
		assert.Equal(t, "blogs;id_blob_id="+hex.EncodeToString([]byte("MarshalBinary(bid)"))+";id_int=1;id_string=str;id_text_id=MarshalText(tid)", key)
	})
	t.Run("LoadOf", func(t *testing.T) {
		meta := meta.LoadOf(new(Blog))
		typ := reflect.TypeOf(Blog{})
		assert.Equal(t, &metaTable{
			Typ:       typ,
			TableName: "blogs",
			PrimaryKey: []*metaColumn{
				&metaColumn{
					Field:      mustFieldByName(typ, "IDInt"),
					ColumnName: "id_int",
				},
				&metaColumn{
					Field:      mustFieldByName(typ, "IDString"),
					ColumnName: "id_string",
				},
				&metaColumn{
					Field:      mustFieldByName(typ, "IDTextID"),
					ColumnName: "id_text_id",
				},
				&metaColumn{
					Field:      mustFieldByName(typ, "IDBlobID"),
					ColumnName: "id_blob_id",
				},
			},
			Columns: []*metaColumn{
				&metaColumn{
					Field:      mustFieldByName(typ, "IDInt"),
					ColumnName: "id_int",
				},
				&metaColumn{
					Field:      mustFieldByName(typ, "IDString"),
					ColumnName: "id_string",
				},
				&metaColumn{
					Field:      mustFieldByName(typ, "IDTextID"),
					ColumnName: "id_text_id",
				},
				&metaColumn{
					Field:      mustFieldByName(typ, "IDBlobID"),
					ColumnName: "id_blob_id",
				},
			},
			RefecenceKeys: [][]*metaColumn{
				[]*metaColumn{
					&metaColumn{
						Field:      mustFieldByName(typ, "IDString"),
						ColumnName: "id_string",
					},
				},
			},
		}, meta)
	})
}
