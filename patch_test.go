package goen

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInsertPatch(t *testing.T) {
	assert.Equal(t, &Patch{
		Kind:      PatchInsert,
		TableName: "testing",
		Columns:   []string{"attr1", "attr2"},
		Values:    []interface{}{1, "2"},
	}, InsertPatch("testing", []string{"attr1", "attr2"}, []interface{}{1, "2"}))

	assert.Panics(t, func() {
		InsertPatch("", []string{"attr1", "attr2"}, []interface{}{1, "2"})
	}, "panics when tableName is empty")

	assert.Panics(t, func() {
		InsertPatch("testing", []string{}, []interface{}{})
	}, "panics when columns and values are empty")

	assert.Panics(t, func() {
		InsertPatch("testing", []string{"attr1", "attr2"}, []interface{}{})
	}, "panics when columns length and values length are mismatched")
}

func TestUpdatePatch(t *testing.T) {
	assert.Equal(t, &Patch{
		Kind:      PatchUpdate,
		TableName: "testing",
		Columns:   []string{"attr1", "attr2"},
		Values:    []interface{}{1, "2"},
		RowKey: &MapRowKey{
			Table: "testing",
			Key: map[string]interface{}{
				"attr3": 9999,
			},
		},
	}, UpdatePatch("testing", []string{"attr1", "attr2"}, []interface{}{1, "2"}, &MapRowKey{
		Table: "testing",
		Key: map[string]interface{}{
			"attr3": 9999,
		},
	}))

	assert.Panics(t, func() {
		UpdatePatch("", []string{"attr1", "attr2"}, []interface{}{1, "2"}, &MapRowKey{
			Table: "testing",
			Key: map[string]interface{}{
				"attr3": 9999,
			},
		})
	}, "panics when tableName is empty")

	assert.Panics(t, func() {
		UpdatePatch("testing", []string{}, []interface{}{}, &MapRowKey{
			Table: "testing",
			Key: map[string]interface{}{
				"attr3": 9999,
			},
		})
	}, "panics when columns and values are empty")

	assert.Panics(t, func() {
		UpdatePatch("testing", []string{"attr1", "attr2"}, []interface{}{}, &MapRowKey{
			Table: "testing",
			Key: map[string]interface{}{
				"attr3": 9999,
			},
		})
	}, "panics when columns length and values length are mismatched")
}

func TestDeletePatch(t *testing.T) {
	assert.Equal(t, &Patch{
		Kind:      PatchDelete,
		TableName: "testing",
		RowKey: &MapRowKey{
			Table: "testing",
			Key: map[string]interface{}{
				"attr3": 9999,
			},
		},
	}, DeletePatch("testing", &MapRowKey{
		Table: "testing",
		Key: map[string]interface{}{
			"attr3": 9999,
		},
	}))

	assert.Panics(t, func() {
		DeletePatch("", &MapRowKey{
			Table: "testing",
			Key: map[string]interface{}{
				"attr3": 9999,
			},
		})
	}, "panics when tableName is empty")
}
