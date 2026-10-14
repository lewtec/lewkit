//go:build windows

package gotool

import (
	"errors"
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

var (
	ErrJitExecutableMappingIsUnavailable3 = errors.New("jit: executable mapping is unavailable")
	ErrJitVirtualprotectFailed            = errors.New("jit: VirtualProtect failed")
)

func prepareWrite([]byte) error { return nil }

func protectRX(mem []byte) error {
	if len(mem) == 0 {
		return nil
	}
	addr := uintptr(unsafe.Pointer(&mem[0]))
	var old uint32
	r, _, _ := procVirtualProtect.Call(addr, uintptr(len(mem)), pageExecuteRead, uintptr(unsafe.Pointer(&old)))
	if r == 0 {
		return ErrJitVirtualprotectFailed
	}
	proc, _, _ := procGetCurrentProcess.Call()
	procFlushInstructionCache.Call(proc, addr, uintptr(len(mem)))
	return nil
}

func mapNear(n int, hint uintptr) ([]byte, error) {
	page := uintptr(os.Getpagesize())
	size := (uintptr(n) + page - 1) &^ (page - 1)
	const granularity = 64 << 10
	try := func(addr uintptr) ([]byte, bool) {
		if addr != 0 {
			addr &^= granularity - 1
			if addr < granularity || addr+size < addr {
				return nil, false
			}
		}
		p, _, _ := procVirtualAlloc.Call(addr, size, memCommit|memReserve, pageReadWrite)
		if p == 0 {
			return nil, false
		}
		if addr != 0 && p != addr {
			procVirtualFree.Call(p, 0, memRelease)
			return nil, false
		}
		return unsafe.Slice((*byte)(unsafe.Pointer(p)), size), true
	}
	limit := uintptr(1 << 31)
	if runtime.GOARCH == "arm64" {
		limit = 96 << 20
	}
	const step = 2 << 20
	base := hint &^ (granularity - 1)
	for dist := uintptr(0); dist < limit; dist += step {
		cands := []uintptr{base + dist}
		if dist > 0 && base > dist {
			cands = append(cands, base-dist)
		}
		for _, addr := range cands {
			if b, ok := try(addr); ok {
				return b, nil
			}
		}
	}
	if b, ok := try(0); ok {
		return b, nil
	}
	return nil, ErrJitExecutableMappingIsUnavailable3
}

const (
	memCommit       = 0x1000
	memReserve      = 0x2000
	memRelease      = 0x8000
	pageReadWrite   = 0x04
	pageExecuteRead = 0x20
)

var (
	kernel32                  = syscall.NewLazyDLL("kernel32.dll")
	procVirtualAlloc          = kernel32.NewProc("VirtualAlloc")
	procVirtualProtect        = kernel32.NewProc("VirtualProtect")
	procVirtualFree           = kernel32.NewProc("VirtualFree")
	procFlushInstructionCache = kernel32.NewProc("FlushInstructionCache")
	procGetCurrentProcess     = kernel32.NewProc("GetCurrentProcess")
)
