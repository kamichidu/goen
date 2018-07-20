package goen

import (
	"container/list"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIncludeBuffer(t *testing.T) {
	t.Run("AddRecords", func(t *testing.T) {
		l := list.New()
		assert.Equal(t, 0, l.Len())

		later := (*IncludeBuffer)(l)
		for _, arg := range []interface{}{
			nil,
			"",
			0,
			true,
			[0]interface{}{},
		} {
			assert.Panics(t, func() {
				later.AddRecords(arg)
			}, "panics if records is %T", arg)
		}
		later.AddRecords([]interface{}{})

		assert.Equal(t, 1, l.Len())
	})
}

func TestIncludeLoaderFunc(t *testing.T) {
	assert.Implements(
		t,
		(*IncludeLoader)(nil),
		IncludeLoaderFunc(func(l *IncludeBuffer, sc *ScopeCache, v interface{}) error {
			return nil
		}),
	)
}

func TestIncludeLoaderList(t *testing.T) {
	assert.Implements(t, (*IncludeLoader)(nil), IncludeLoaderList(nil))
}
