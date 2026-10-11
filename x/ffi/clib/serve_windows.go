//go:build windows && (amd64 || arm64 || 386)

package clib

import (
	"encoding/binary"
	"sync/atomic"
	"syscall"
	"unsafe"
)

// pipeR and pipeW are kernel32 pipe handles. The entry creates them
// before starting the runtime. rtReady is 0 while that thread is coming
// up, 1 once serve is reading, and 2 when the pipe could not be created.
// booted is 0 until the first caller wins. The assembly loads these words.
var (
	pipeR   uintptr
	pipeW   uintptr
	rtReady uint32
	booted  uint32
)

var procReadFile = syscall.NewLazyDLL("kernel32.dll").NewProc("ReadFile")

func init() {
	r := atomic.LoadUintptr(&pipeR)
	if r == 0 {
		return
	}
	go serve(r)
}

func serve(h uintptr) {
	atomic.StoreUint32(&rtReady, 1)
	var buf [8]byte
	filled := 0
	tmp := make([]byte, 8)
	for {
		n, _ := readFile(h, tmp[:8-filled])
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
		return
	}
}

func readFile(h uintptr, buf []byte) (int, error) {
	if len(buf) == 0 {
		return 0, nil
	}
	var done uint32
	r, _, err := procReadFile.Call(
		h,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
		uintptr(unsafe.Pointer(&done)),
		0,
	)
	if r == 0 {
		return 0, err
	}
	return int(done), nil
}

func wake(*uint32) {}
