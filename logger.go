package goen

import (
	"fmt"
)

type Logger interface {
	Print(...interface{})
	Printf(string, ...interface{})
}

type LoggerFunc func(...interface{})

func (fn LoggerFunc) Print(args ...interface{}) {
	fn(args...)
}

type stringerFunc func() string

func (fn stringerFunc) String() string {
	return fn()
}

func (fn LoggerFunc) Printf(format string, args ...interface{}) {
	// for lazy eval
	fn(stringerFunc(func() string {
		return fmt.Sprintf(format, args...)
	}))
}
