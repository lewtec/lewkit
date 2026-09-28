//go:build windows

package native

import "syscall"

// Call invokes the procedure. The first call loads the library.
func (p Proc) Call(args ...uintptr) (uintptr, uintptr, error) {
	addr, err := p.load()
	if err != nil {
		return 0, 0, err
	}
	return syscall.SyscallN(addr, args...)
}
