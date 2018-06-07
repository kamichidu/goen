package goen

import (
	"container/list"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIncludeLoaderFunc(t *testing.T) {
	assert.Implements(
		t,
		(*IncludeLoader)(nil),
		IncludeLoaderFunc(func(l *list.List, sc *ScopeCache, v interface{}) error {
			return nil
		}),
	)
}

func TestIncludeLoaderList(t *testing.T) {
	assert.Implements(t, (*IncludeLoader)(nil), IncludeLoaderList(nil))
}
