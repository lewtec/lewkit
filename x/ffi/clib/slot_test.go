//go:build ((linux || android) && (amd64 || arm64 || 386 || arm)) || (windows && (amd64 || arm64 || 386)) || (darwin && !ios && (amd64 || arm64))

package clib

import (
	"fmt"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
)

func TestSlotLayout(t *testing.T) {
	var slot Slot
	if unsafe.Sizeof(slot) != 160 {
		require.FailNowf(t, "", "size %d", unsafe.Sizeof(slot))
	}
	if unsafe.Offsetof(slot.Args) != 8 {
		require.FailNowf(t, "", "args %d", unsafe.Offsetof(slot.Args))
	}
	if unsafe.Offsetof(slot.XMM) != 72 {
		require.FailNowf(t, "", "xmm %d", unsafe.Offsetof(slot.XMM))
	}
	if unsafe.Offsetof(slot.Ret) != 136 {
		require.FailNowf(t, "", "ret %d", unsafe.Offsetof(slot.Ret))
	}
	if unsafe.Offsetof(slot.RetX) != 144 {
		require.FailNowf(t, "", "retx %d", unsafe.Offsetof(slot.RetX))
	}
	if unsafe.Offsetof(slot.Done) != 152 {
		require.FailNowf(t, "", "done %d", unsafe.Offsetof(slot.Done))
	}
	if unsafe.Offsetof(slot.Kind) != 156 {
		require.FailNowf(t, "", "kind %d", unsafe.Offsetof(slot.Kind))
	}
}

func TestHandleRejectsNegative(t *testing.T) {
	defer func() {
		if recover() == nil {
			require.FailNow(t, fmt.Sprint("expected panic"))
		}
	}()
	Handle(-1, nil)
}

func TestU32CrossesWord(t *testing.T) {
	args := [8]uint64{0x0807060504030201, 0xffeeddccbbaa9988}
	if U32(&args, 0) != 0x04030201 {
		require.FailNowf(t, "", "low %x", U32(&args, 0))
	}
	if U32(&args, 4) != 0x08070605 {
		require.FailNowf(t, "", "mid %x", U32(&args, 4))
	}
	if U64(&args, 4) != 0xbbaa998808070605 {
		require.FailNowf(t, "", "wide %x", U64(&args, 4))
	}
	if U32(nil, 0) != 0 || U64(&args, 80) != 0 {
		require.FailNow(t, fmt.Sprint("bounds"))
	}
}
