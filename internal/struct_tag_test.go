package internal_test

import (
	"reflect"
	"testing"

	"github.com/kamichidu/goen/internal"
	"github.com/stretchr/testify/assert"
)

func TestFirstLookup(t *testing.T) {
	cases := []struct {
		Tag    string
		Names  []string
		Expect string
	}{
		{``, []string{}, ""},
		{`a:"hoge"`, []string{"a"}, "hoge"},
		{`a:"hoge" b:"fuga"`, []string{"b"}, "fuga"},
	}
	for _, c := range cases {
		val, ok := internal.FirstLookup(reflect.StructTag(c.Tag), c.Names...)
		assert.Equal(t, c.Expect != "", ok, "tag=%q names=%q", c.Tag, c.Names)
		assert.Equal(t, c.Expect, val, "tag=%q names=%q", c.Tag, c.Names)
	}
}
