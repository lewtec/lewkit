package clib

import (
	"fmt"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
)

func TestStoreLoadStruct(t *testing.T) {
	type tally struct {
		sum     int64
		workers int32
		_       int32
	}
	var got tally
	Store(unsafe.Pointer(&got), int(unsafe.Sizeof(got)), 0, 42)
	Store(unsafe.Pointer(&got), int(unsafe.Sizeof(got)), 8, 3)
	if got.sum != 42 || got.workers != 3 {
		require.FailNowf(t, "", "store %+v", got)
	}
	if Load(unsafe.Pointer(&got), int(unsafe.Sizeof(got)), 0) != 42 {
		require.FailNow(t, fmt.Sprint("load low"))
	}
	if Load(unsafe.Pointer(&got), int(unsafe.Sizeof(got)), 8) != 3 {
		require.FailNow(t, fmt.Sprint("load high"))
	}
	if Load(unsafe.Pointer(&got), int(unsafe.Sizeof(got)), 16) != 0 {
		require.FailNow(t, fmt.Sprint("load past end"))
	}
}
