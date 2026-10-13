package clib

import (
	"testing"
	"unsafe"
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
		t.Fatalf("store %+v", got)
	}
	if Load(unsafe.Pointer(&got), int(unsafe.Sizeof(got)), 0) != 42 {
		t.Fatal("load low")
	}
	if Load(unsafe.Pointer(&got), int(unsafe.Sizeof(got)), 8) != 3 {
		t.Fatal("load high")
	}
	if Load(unsafe.Pointer(&got), int(unsafe.Sizeof(got)), 16) != 0 {
		t.Fatal("load past end")
	}
}
