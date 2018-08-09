// +build testdata

package testing

import (
	"time"
)

type Record struct {
	Value *time.Time `goen:""`
}
