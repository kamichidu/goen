// +build testdata

package testing

import (
	"github.com/satori/go.uuid"
	"time"
)

type DeclTypes struct {
	StringDecl         string
	IntDecl            int
	StringPtrDecl      *string
	StringSliceDecl    []string
	StringPtrSliceDecl []*string
	StringSlicePtrDecl *[]string
	TimeDecl           time.Time
	TimePtrDecl        *time.Time
	TimeSliceDecl      []time.Time
	TimePtrSliceDecl   []*time.Time
	TimeSlicePtrDecl   *[]time.Time
}

type DeclPkgPaths struct {
	NoPkgPath string
	PkgPath   uuid.UUID
}
