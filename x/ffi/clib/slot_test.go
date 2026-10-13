//go:build ((linux || android) && (amd64 || arm64 || 386 || arm)) || (windows && (amd64 || arm64 || 386)) || (darwin && !ios && (amd64 || arm64))

package clib

import (
	"testing"
	"unsafe"
)

func TestSlotLayout(t *testing.T) {
	var slot Slot
	if unsafe.Sizeof(slot) != 160 {
		t.Fatalf("size %d", unsafe.Sizeof(slot))
	}
	if unsafe.Offsetof(slot.Args) != 8 {
		t.Fatalf("args %d", unsafe.Offsetof(slot.Args))
	}
	if unsafe.Offsetof(slot.XMM) != 72 {
		t.Fatalf("xmm %d", unsafe.Offsetof(slot.XMM))
	}
	if unsafe.Offsetof(slot.Ret) != 136 {
		t.Fatalf("ret %d", unsafe.Offsetof(slot.Ret))
	}
	if unsafe.Offsetof(slot.RetX) != 144 {
		t.Fatalf("retx %d", unsafe.Offsetof(slot.RetX))
	}
	if unsafe.Offsetof(slot.Done) != 152 {
		t.Fatalf("done %d", unsafe.Offsetof(slot.Done))
	}
	if unsafe.Offsetof(slot.Kind) != 156 {
		t.Fatalf("kind %d", unsafe.Offsetof(slot.Kind))
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

func TestU32CrossesWord(t *testing.T) {
	args := [8]uint64{0x0807060504030201, 0xffeeddccbbaa9988}
	if U32(&args, 0) != 0x04030201 {
		t.Fatalf("low %x", U32(&args, 0))
	}
	if U32(&args, 4) != 0x08070605 {
		t.Fatalf("mid %x", U32(&args, 4))
	}
	if U64(&args, 4) != 0xbbaa998808070605 {
		t.Fatalf("wide %x", U64(&args, 4))
	}
	if U32(nil, 0) != 0 || U64(&args, 80) != 0 {
		t.Fatal("bounds")
	}
}
