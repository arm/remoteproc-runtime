package e2e

import (
	"sync/atomic"
)

var testCounter atomic.Uint32

func getTestNumber() uint {
	return uint(testCounter.Add(1))
}
