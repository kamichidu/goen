package goen

import (
	"container/list"
)

//go:generate go run genlist.go -o utils_gen.go

func NewPatchList() *PatchList {
	return (*PatchList)(list.New())
}

func NewSqlizerList() *SqlizerList {
	return (*SqlizerList)(list.New())
}
