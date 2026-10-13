//go:build windows

package purego

import "syscall"

// Open loads a DLL. path is a file name such as kernel32.dll.
func Open(path string) (uintptr, error) {
	dll, err := syscall.LoadDLL(path)
	if err != nil {
		return 0, err
	}
	return uintptr(dll.Handle), nil
}
