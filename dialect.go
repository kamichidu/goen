package goen

import (
	"sync"

	"github.com/kamichidu/goen/dialect"
)

var (
	dialectsMu sync.RWMutex
	dialects   = make(map[string]dialect.Dialect)
)

func Register(name string, dialect dialect.Dialect) {
	dialectsMu.Lock()
	defer dialectsMu.Unlock()

	if _, dup := dialects[name]; dup {
		panic("goen: Register called twice for dialect " + name)
	}
	dialects[name] = dialect
}
