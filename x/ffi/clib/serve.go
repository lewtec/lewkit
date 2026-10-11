//go:build (linux || android) && (amd64 || arm64 || 386 || arm)

package clib

import (
	"encoding/binary"
	"sync/atomic"
	"syscall"
	"unsafe"
)

const (
	futexWaitPrivate = 128
	futexWakePrivate = 129
)

// pipeW is the write end of the call pipe.
// rtReady is 0 while the runtime is coming up, 1 once serve is reading,
// and 2 when the clone failed. booted is 0 until the first caller wins.
// The assembly entry loads these words. They stay naturally aligned.
var (
	pipeW   int32
	rtReady uint32
	booted  uint32
)

func init() {
	var fds [2]int
	if err := syscall.Pipe(fds[:]); err != nil {
		panic(err)
	}
	atomic.StoreInt32(&pipeW, int32(fds[1]))
	go serve(fds[0])
}

// serve reads one slot pointer at a time and runs it on its own goroutine.
// A direct call would deadlock when the handler calls back into C and C
// calls back into the library: the reader would still be inside the first
// call. The ready flag is stored before the read so a caller that woke
// on it can write a pointer the pipe buffer holds until Read.
// The pointer is eight bytes on every arch. A 32-bit caller zero-extends it.
func serve(r int) {
	atomic.StoreUint32(&rtReady, 1)
	wake(&rtReady)
	var buf [8]byte
	filled := 0
	tmp := make([]byte, 8)
	for {
		n, err := syscall.Read(r, tmp[:8-filled])
		if n > 0 {
			copy(buf[filled:], tmp[:n])
			filled += n
			if filled < 8 {
				continue
			}
			filled = 0
			ptr := binary.LittleEndian.Uint64(buf[:])
			if ptr == 0 {
				continue
			}
			go dispatch((*Slot)(Ptr(uintptr(ptr))))
			continue
		}
		if err == syscall.EINTR {
			continue
		}
		return
	}
}

func dispatch(slot *Slot) {
	defer func() {
		if recover() != nil {
			slot.Ret = 0
			slot.RetX = 0
			slot.Kind = KindInt
		}
		atomic.StoreUint32(&slot.Done, 1)
		wake(&slot.Done)
	}()
	h := handler(slot.ID)
	if h == nil {
		return
	}
	slot.Ret, slot.RetX, slot.Kind = h(&slot.Args, &slot.XMM)
}

func wake(addr *uint32) {
	_, _, _ = syscall.Syscall6(syscall.SYS_FUTEX, uintptr(unsafe.Pointer(addr)), uintptr(futexWakePrivate), 1, 0, 0, 0)
}
