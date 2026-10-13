//go:build linux

package gotool

import (
	"unsafe"

	"golang.org/x/sys/unix"
)

func prepareWrite([]byte) error { return nil }

func protectRX(mem []byte) error {
	return unix.Mprotect(mem, unix.PROT_READ|unix.PROT_EXEC)
}

func mapNear(n int, hint uintptr) ([]byte, error) {
	page := uintptr(unix.Getpagesize())
	size := (uintptr(n) + page - 1) &^ (page - 1)
	prot := uintptr(unix.PROT_READ | unix.PROT_WRITE)
	flags := uintptr(unix.MAP_PRIVATE | unix.MAP_ANONYMOUS | unix.MAP_FIXED_NOREPLACE)
	try := func(addr uintptr) ([]byte, bool) {
		if addr < page || addr+size < addr {
			return nil, false
		}
		p, _, e := unix.Syscall6(unix.SYS_MMAP, addr, size, prot, flags, ^uintptr(0), 0)
		if e != 0 {
			return nil, false
		}
		return unsafe.Slice((*byte)(unsafe.Pointer(p)), size), true
	}
	const step = 2 << 20
	const limit = uintptr(1 << 31)
	base := hint &^ (page - 1)
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
	b, err := unix.Mmap(-1, 0, int(size), unix.PROT_READ|unix.PROT_WRITE, unix.MAP_PRIVATE|unix.MAP_ANONYMOUS)
	if err != nil {
		return nil, err
	}
	return b, nil
}
