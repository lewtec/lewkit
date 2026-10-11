//go:build darwin && !ios && (amd64 || arm64)

package clib

import (
	"encoding/binary"
	"sync/atomic"
	"syscall"
)

// pipeR and pipeW are the libSystem pipe fds stored as words.
// The entry creates them before starting the runtime. rtReady is 0 while
// that thread is coming up, 1 once serve is reading, and 2 when pipe failed.
// booted is 0 until the first caller wins. The assembly loads these words.
var (
	pipeR   uintptr
	pipeW   uintptr
	rtReady uint32
	booted  uint32
)

func init() {
	r := atomic.LoadUintptr(&pipeR)
	if r == 0 {
		return
	}
	go serve(int(r))
}

func serve(r int) {
	atomic.StoreUint32(&rtReady, 1)
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

func wake(*uint32) {}
