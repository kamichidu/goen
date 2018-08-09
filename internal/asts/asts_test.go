package asts

import (
	"go/ast"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVisitorFunc(t *testing.T) {
	assert.Implements(t, (*ast.Visitor)(nil), VisitorFunc(func(_ ast.Node) ast.Visitor { return nil }))
}
