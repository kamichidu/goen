package goen

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSafeDo(t *testing.T) {
	assert.NotPanics(t, func() {
		var err, inerr error
		var val interface{}

		err = safeDo(func() {
			val = "inside fn"
		})
		assert.NoError(t, err)
		assert.Equal(t, "inside fn", val)

		err = safeDo(func() {
			panic("suppress panic")
		})
		assert.Equal(t, errors.New("PANIC=suppress panic"), err)

		err = safeDo(func() {
			inerr = errors.New("returns error")
		})
		assert.NoError(t, err)
		assert.Equal(t, errors.New("returns error"), inerr)
	})
}
