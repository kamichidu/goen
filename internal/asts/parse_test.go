package asts

import (
	"testing"

	_ "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestParsePkgPath(t *testing.T) {
	cases := []struct {
		PkgName string
		PkgPath string
	}{
		{"reflect", "reflect"},
		{"ast", "go/ast"},
		{"uuid", "github.com/satori/go.uuid"},
	}
	for _, c := range cases {
		pkg, err := ParsePkgPath(c.PkgPath)
		if !assert.NoError(t, err) {
			continue
		}
		assert.Equal(t, c.PkgName, pkg.Name)
	}
}
