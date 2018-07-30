package goen

import (
	"container/list"
	"fmt"
)

//go:generate go run tools/genlist.go -o utils_gen.go

func NewPatchList() *PatchList {
	return (*PatchList)(list.New())
}

func NewSqlizerList() *SqlizerList {
	return (*SqlizerList)(list.New())
}

// safeDo is a utility func to invoke fn with recover.
// when panic inside fn, it returns err; or nil.
func safeDo(fn func()) (err error) {
	defer func() {
		if perr := recover(); perr != nil {
			switch v := perr.(type) {
			case error:
				err = v
			default:
				err = fmt.Errorf("PANIC=%v", perr)
			}
		}
	}()
	fn()
	return nil
}
