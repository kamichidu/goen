package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var d = &dialect{}

func TestDialect(t *testing.T) {
	t.Run("Quote", func(t *testing.T) {
		cases := []struct {
			S string
			R string
		}{
			{"hoge", `"hoge"`},
			{"ho`ge", "\"ho`ge\""},
			{`ho"ge`, `"ho""ge"`},
			{`ho""ge`, `"ho""""ge"`},
		}
		for _, c := range cases {
			assert.Equal(t, c.R, d.Quote(c.S))
		}
	})
}
