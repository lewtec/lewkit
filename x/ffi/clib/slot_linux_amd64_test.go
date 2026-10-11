//go:build linux && amd64

package clib

import (
	"testing"
	"unsafe"
)

func TestSlotLayout(t *testing.T) {
	var slot Slot
	if unsafe.Sizeof(slot) != 144 {
		t.Fatalf("size %d", unsafe.Sizeof(slot))
	}
	if unsafe.Offsetof(slot.Args) != 8 {
		t.Fatalf("args %d", unsafe.Offsetof(slot.Args))
	}
	if unsafe.Offsetof(slot.XMM) != 56 {
		t.Fatalf("xmm %d", unsafe.Offsetof(slot.XMM))
	}
	if unsafe.Offsetof(slot.Ret) != 120 {
		t.Fatalf("ret %d", unsafe.Offsetof(slot.Ret))
	}
	if unsafe.Offsetof(slot.RetX) != 128 {
		t.Fatalf("retx %d", unsafe.Offsetof(slot.RetX))
	}
	if unsafe.Offsetof(slot.Done) != 136 {
		t.Fatalf("done %d", unsafe.Offsetof(slot.Done))
	}
}

func TestHandleRejectsNegative(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	Handle(-1, nil)
}
