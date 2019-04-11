package goen

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoggerFunc(t *testing.T) {
	var buffer bytes.Buffer
	var logger Logger = LoggerFunc(func(args ...interface{}) {
		fmt.Fprint(&buffer, args...)
	})
	assert.Empty(t, buffer.String())
	logger.Print("hello\n")
	logger.Print("world\n")
	assert.Equal(t, "hello\nworld\n", buffer.String())
	buffer.Reset()
	assert.Empty(t, buffer.String())
	logger.Printf("i=%d\n", 0)
	logger.Printf("i=%d\n", 1)
	assert.Equal(t, "i=0\ni=1\n", buffer.String())
}
