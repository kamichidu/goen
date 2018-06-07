package main

import (
	"fmt"
	"go/format"

	"golang.org/x/tools/imports"
)

type goFmtFilter struct{}

func (*goFmtFilter) Filter(filename string, src []byte) ([]byte, error) {
	res, err := format.Source(src)
	if err != nil {
		return nil, fmt.Errorf("formatting go files error: %s", err)
	}
	return res, nil
}

type goImportsFilter struct{}

func (*goImportsFilter) Filter(filename string, src []byte) ([]byte, error) {
	res, err := imports.Process(filename, src, &imports.Options{
		AllErrors:  true,
		FormatOnly: true,
		Comments:   true,
	})
	if err != nil {
		return nil, fmt.Errorf("formatting go imports error: %s", err)
	}
	return res, nil
}
