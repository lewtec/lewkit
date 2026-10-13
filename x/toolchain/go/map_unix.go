//go:build unix && !linux && !darwin

package gotool

import "golang.org/x/sys/unix"

func prepareWrite([]byte) error { return nil }

func protectRX(mem []byte) error {
	return unix.Mprotect(mem, unix.PROT_READ|unix.PROT_EXEC)
}

func mapNear(n int, _ uintptr) ([]byte, error) {
	page := unix.Getpagesize()
	size := (n + page - 1) &^ (page - 1)
	return unix.Mmap(-1, 0, size, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_PRIVATE|unix.MAP_ANONYMOUS)
}
