package multiref

import (
	uuid "github.com/satori/go.uuid"
)

//go:generate goen -o goen.go
type Parent struct {
	ParentID uuid.UUID `goen:"" table:"parent" primary_key:""`

	GroupID uuid.UUID

	Children []*Child `foreign_key:"parent_id,group_id"`
}

type Child struct {
	ChildID uuid.UUID `goen:"" table:"child" primary_key:""`

	ParentID uuid.UUID

	GroupID uuid.UUID

	Parent *Parent `foreign_key:"parent_id,group_id"`
}
