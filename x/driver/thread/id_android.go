//go:build android

package thread

import "syscall"

// osThread is the kernel thread id. Android has no libc.so.6, and dlopen
// from this process deadlocks the linker, so pthread_self stays on desktop Unix.
func osThread() uint64 {
	id, _, errno := syscall.RawSyscall(syscall.SYS_GETTID, 0, 0, 0)
	if errno != 0 || id == 0 {
		return 0
	}
	return uint64(id)
}
