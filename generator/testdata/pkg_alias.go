// +build testdata

package testing

import (
	alias "time"

	"github.com/satori/go.uuid"
	uuid2 "github.com/satori/go.uuid"
)

type Record struct {
	Value *alias.Time `goen:""`

	Value2 uuid.UUID

	Value3 uuid2.UUID
}
