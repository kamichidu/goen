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
