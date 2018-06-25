package goen

type PatchKind int

const (
	PatchInsert PatchKind = iota
	PatchUpdate
	PatchDelete
)

type Patch struct {
	Kind PatchKind

	TableName string

	RowKey RowKey

	Columns []string

	Values []interface{}
}

func InsertPatch(tableName string, columns []string, values []interface{}) *Patch {
	if tableName == "" {
		panic("goen: no tableName provided")
	}
	if len(columns) == 0 || len(columns) != len(values) {
		panic("goen: columns and values must have least 1 element or length mismatched")
	}

	return &Patch{
		Kind:      PatchInsert,
		TableName: tableName,
		Columns:   columns,
		Values:    values,
	}
}

func UpdatePatch(tableName string, columns []string, values []interface{}, rowKey RowKey) *Patch {
	if tableName == "" {
		panic("goen: no tableName provided")
	}
	if len(columns) == 0 || len(columns) != len(values) {
		panic("goen: columns and values must have least 1 element or length mismatched")
	}

	return &Patch{
		Kind:      PatchUpdate,
		TableName: tableName,
		Columns:   columns,
		Values:    values,
		RowKey:    rowKey,
	}
}

func DeletePatch(tableName string, rowKey RowKey) *Patch {
	if tableName == "" {
		panic("goen: no tableName provided")
	}

	return &Patch{
		Kind:      PatchDelete,
		TableName: tableName,
		RowKey:    rowKey,
	}
}
