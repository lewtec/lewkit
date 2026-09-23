//go:build windows

package native

import (
	"fmt"
	"syscall"
)

const (
	Lazy   = 0
	Now    = 0
	Global = 0
	Local  = 0
)

// Open loads path with LoadLibrary. flags are ignored.
func Open(path string, flags int) (uintptr, error) {
	_ = flags
	dll, err := syscall.LoadDLL(path)
	if err != nil {
		return 0, fmt.Errorf("load %s: %w", path, err)
	}
	return uintptr(dll.Handle), nil
}

// Symbol looks up name in a library opened by Open.
func Symbol(lib uintptr, name string) (uintptr, error) {
	proc, err := (&syscall.DLL{Handle: syscall.Handle(lib)}).FindProc(name)
	if err != nil {
		return 0, err
	}
	return proc.Addr(), nil
}
