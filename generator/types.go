package generator

type Import struct {
	Name string

	Path string
}

type Package struct {
	PackageName string

	Imports []*Import

	Tables []*Table
}

type Table struct {
	TableName string

	ReadOnly bool

	Entity string

	Columns []*Column

	OneToManyRelations []*Relation

	ManyToOneRelations []*Relation

	OneToOneRelations []*Relation
}

type Column struct {
	ColumnName string

	IsPK bool

	OmitEmpty bool

	FieldName string

	FieldType string
}

// Relation represents another table relation information.
type Relation struct {
	// another table name
	TableName string

	// this entity's field name
	FieldName string

	// another table entity type
	FieldType string

	// another table column names
	ColumnNames []string

	// this table columns
	ForeignKeys []*RelationalColumn

	// another table columns
	References []*RelationalColumn
}

type RelationalColumn struct {
	ColumnName string

	FieldName string

	FieldType string
}
