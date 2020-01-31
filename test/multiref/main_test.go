package multiref_test

import (
	"testing"

	"github.com/kamichidu/goen/test/multiref"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestMultiRef(t *testing.T) {
	dbc := multiref.NewDBContext(prepareDB())
	dbc.DebugMode(true)

	parent := &multiref.Parent{
		ParentID: uuid.Must(uuid.NewV4()),
		GroupID:  uuid.Must(uuid.NewV4()),
	}
	dbc.Parent.Insert(parent)

	children := []*multiref.Child{
		&multiref.Child{
			ChildID:  uuid.Must(uuid.NewV4()),
			ParentID: parent.ParentID,
			GroupID:  parent.GroupID,
		},
		&multiref.Child{
			ChildID:  uuid.Must(uuid.NewV4()),
			ParentID: parent.ParentID,
			GroupID:  parent.GroupID,
		},
	}
	for _, child := range children {
		dbc.Child.Insert(child)
	}

	if err := dbc.SaveChanges(); err != nil {
		panic(err)
	}

	t.Run("one to many relation", func(t *testing.T) {
		parent, err := dbc.Parent.Select().Include(
			dbc.Parent.IncludeChildren,
		).QueryRow()
		if assert.NoError(t, err) {
			assert.Len(t, parent.Children, 2, "include loader loads an entity by multi column reference")
			assert.Equal(t, children, parent.Children, "include loader loads an expected entity by multi column reference")
		}
	})
	t.Run("many to one relation", func(t *testing.T) {
		child, err := dbc.Child.Select().Include(
			dbc.Child.IncludeParent,
		).Where(dbc.Child.ChildID.Eq(children[0].ChildID)).QueryRow()
		if assert.NoError(t, err) {
			assert.Equal(t, parent, child.Parent, "include loader loads an expected entity by multi column reference")
		}
	})
}
