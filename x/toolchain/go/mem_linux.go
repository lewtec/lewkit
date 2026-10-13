//go:build linux

package gotool

import (
	"os"
	"strconv"

	"golang.org/x/sys/unix"
)

func openMem(name string) (*memFile, error) {
	fd, err := unix.MemfdCreate(name, 0)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), name)
	path := "/proc/" + strconv.Itoa(os.Getpid()) + "/fd/" + strconv.Itoa(fd)
	return &memFile{f: f, path: path}, nil
}
